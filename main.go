package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var (
	currentCallerName   string
	currentFunctionName = "LABEL"
	retIndex            = 1
)

type CommandType int
type SegmentType int
type ALType int

const (
	CommandTypeArithmetic CommandType = iota
	CommandTypePush
	CommandTypePop
	CommandTypeLabel
	CommandTypeGOTO
	CommandTypeIf
	CommandTypeFunction
	CommandTypeReturn
	CommandTypeCall
)

func (ct CommandType) String() string {
	return []string{
		"arithmetic",
		"push",
		"pop",
		"label",
		"goto",
		"if-goto",
		"function",
		"return",
		"call",
	}[ct]
}

const (
	SegmentTypeConstant SegmentType = iota
	SegmentTypeLocal
	SegmentTypeArgument
	SegmentTypeThis
	SegmentTypeThat
	SegmentTypeStatic
	SegmentTypeTemp
	SegmentTypePointer
)

func (st SegmentType) String() string {
	return []string{
		"constant",
		"local",
		"argument",
		"this",
		"that",
		"static",
		"temp",
		"pointer",
	}[st]
}

func (st SegmentType) ID() string {
	return []string{
		"",     // constant
		"LCL",  // local
		"ARG",  // argument
		"THIS", // this
		"THAT", // that
		"",     // static
		"",     // temp
		"",     // pointer
	}[st]
}

const (
	ALTypeAdd ALType = iota
	ALTypeSub
	ALTypeNeg
	ALTypeEq
	ALTypeGt
	ALTypeLt
	ALTypeAnd
	ALTypeOr
	ALTypeNot
)

func (lt ALType) String() string {
	return []string{
		"add",
		"sub",
		"neg",
		"eq",
		"gt",
		"lt",
		"and",
		"or",
		"not",
	}[lt]
}

func main() {
	var vmSrcFiles, cmpFile string
	flag.StringVar(&vmSrcFiles, "s", "", "source file in vm extension (e.g. Add.vm or a Directory with multiple vm files)")
	flag.StringVar(&cmpFile, "c", "", "compare file")
	flag.Parse()
	if vmSrcFiles == "" {
		fmt.Println("No source file provided")
		flag.Usage()
		os.Exit(1)
	}
	// check if the source is a directory
	srcStat, err := os.Stat(vmSrcFiles)
	if err != nil {
		fmt.Println("Error getting source file status", err)
		os.Exit(1)
	}

	dstFile := ""
	srcFiles := []*os.File{}
	defer func() {
		for _, f := range srcFiles {
			f.Close()
		}
	}()

	if srcStat.IsDir() {
		basename := filepath.Base(vmSrcFiles)
		dstFile = filepath.Join(vmSrcFiles, basename+".asm")

		files, err := filepath.Glob(filepath.Join(vmSrcFiles, "*.vm"))
		if err != nil {
			fmt.Printf("Error listing files %s: %s\n", vmSrcFiles, err)
			os.Exit(1)
		}
		for _, file := range files {
			srcF, err := os.Open(file)
			if err != nil {
				fmt.Printf("Error opening source file %s: %s\n", file, err)
				os.Exit(1)
			}
			srcFiles = append(srcFiles, srcF)
		}
	} else {
		dstFile = strings.TrimSuffix(filepath.Base(vmSrcFiles), filepath.Ext(vmSrcFiles))
		dstFile = filepath.Join(filepath.Dir(vmSrcFiles), dstFile+".asm")

		srcF, err := os.Open(vmSrcFiles)
		if err != nil {
			fmt.Printf("Error opening source file %s: %s\n", vmSrcFiles, err)
			os.Exit(1)
		}
		srcFiles = append(srcFiles, srcF)
	}

	hasMultipleSrcFiles := len(srcFiles) > 1
	var fileWithSysInit *os.File

	for _, srcF := range srcFiles {
		scanner := bufio.NewScanner(srcF)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "//") ||
				strings.TrimSpace(line) == "" {
				continue
			}
			if strings.Contains(line, "function Sys.init 0") {
				fileWithSysInit = srcF
				break
			}
		}
		if fileWithSysInit != nil {
			break
		}
	}
	if fileWithSysInit == nil && hasMultipleSrcFiles {
		fmt.Println("Sys.init not found in any source file")
		os.Exit(1)
	}

	// create dst if not exists
	var dstF *os.File
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		dstF, err = os.Create(dstFile)
		defer dstF.Close()
		if err != nil {
			fmt.Println("Error creating destination file", err)
			os.Exit(1)
		}
	} else {
		dstF, err = os.OpenFile(dstFile, os.O_WRONLY, 0644)
		defer dstF.Close()
		if err != nil {
			fmt.Println("Error opening destination file", err)
			os.Exit(1)
		}
	}

	// rawSourceLines := []string{}
	instructionsLines := []string{}

	if hasMultipleSrcFiles {
		// we need to scan the file with Sys.init last
		// Create a new slice with Sys.init file last
		files := make([]*os.File, 0, len(srcFiles))
		// Add all other files first
		for _, f := range srcFiles {
			if f != fileWithSysInit {
				files = append(files, f)
			}
		}
		// Add the file with Sys.init last if it exists
		if fileWithSysInit != nil {
			files = append(files, fileWithSysInit)
		}
		// Loop through all files in the correct order
		for _, sFile := range files {
			// Reset file pointer to beginning of file
			sFile.Seek(0, 0)
			scanner := bufio.NewScanner(sFile)
			for scanner.Scan() {
				line := removeCommentsAndSpaces(scanner.Text())
				if line == "" {
					continue
				}
				line = encodeLineFileName(sFile.Name(), line)
				instructionsLines = append(instructionsLines, line)
			}
			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading file %s: %v\n", sFile.Name(), err)
				os.Exit(1)
			}
		}
	} else {
		sFile := srcFiles[0]
		sFile.Seek(0, 0)
		scanner := bufio.NewScanner(sFile)
		for scanner.Scan() {
			line := removeCommentsAndSpaces(scanner.Text())
			if line == "" {
				continue
			}
			line = encodeLineFileName(sFile.Name(), line)
			instructionsLines = append(instructionsLines, line)
		}
	}

	if len(instructionsLines) == 0 {
		fmt.Println("No source lines found")
		os.Exit(1)
	}

	resultLines := []string{}

	if fileWithSysInit != nil {
		lines := []string{
			"// Bootstrap code",
			"@256",
			"D=A",
			"@SP",
			"M=D",
			"/// call Sys.init 0",
		}
		lines = append(lines, genCall("Sys.init", 0)...)
		resultLines = append(resultLines, lines...)
	}

	for i, rLine := range instructionsLines {
		fileName, line := decodeLineFileName(rLine)
		instruction, err := parseInstruction(i, fileName, line)
		if err != nil {
			fmt.Println("Error parsing instruction", err)
			os.Exit(2)
		}
		asm, err := instruction.GenAsm()
		if err != nil {
			fmt.Println("Error generating asm", err)
			os.Exit(2)
		}
		resultLines = append(resultLines, asm...)
	}

	// MARK: - Write to Destination File
	if err = writeLinesToDst(dstF, resultLines); err != nil {
		fmt.Println("Error writing to destination file", err)
		os.Exit(2)
	}

	fmt.Println("Successfully wrote to destination file:", dstFile)

	// MARK: - Compare with Expected Output
	if cmpFile != "" {
		// read cmp file and compare with filteredLines
		cmpF, err := os.Open(cmpFile)
		if err != nil {
			fmt.Println("Error opening compare file", err)
			os.Exit(2)
		}
		defer cmpF.Close()
		cmpLines := []string{}
		scanner := bufio.NewScanner(cmpF)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			cmpLines = append(cmpLines, line)
		}
		if len(cmpLines) != len(resultLines) {
			fmt.Println("Compare file has a different number of lines than the source file")
			os.Exit(2)
		}
		for i, line := range cmpLines {
			if line != resultLines[i] {
				fmt.Printf(
					"Error in file %s:%d %s\n"+
						"\t Expected: %s\n"+
						"\t Got: %s\n",
					cmpFile,
					i+1,
					"lines are not equal",
					line,
					resultLines[i],
				)
				os.Exit(2)
			}
		}
		fmt.Println("Successfully compared files")
	}

}

func encodeLineFileName(fileName, line string) string {
	fileName = strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	fileName = strings.ReplaceAll(fileName, " ", "_")
	return fmt.Sprintf("%s#%s", fileName, line)
}

func decodeLineFileName(line string) (string, string) {
	parts := strings.Split(line, "#")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}

func removeCommentsAndSpaces(line string) string {
	v := strings.Split(line, "//")
	if len(v) == 0 {
		return ""
	}
	return strings.TrimSpace(v[0])
}

func writeLinesToDst(dstF *os.File, lines []string) error {
	dstF.Truncate(0)
	dstF.Seek(0, 0)
	for _, line := range lines {
		dstF.WriteString(line + "\n")
	}
	return nil
}

type Instruction struct {
	FileName    string
	Line        string
	CommandType CommandType
	Arg1        string
	SegmentType SegmentType
	Arg2        string
	Arg2Val     int
	ALType      ALType
	Index       int
}

func (i *Instruction) String() string {
	if i.CommandType == CommandTypeArithmetic {
		return i.Arg1
	}
	if i.CommandType == CommandTypePush {
		return fmt.Sprintf("push %s %d", i.SegmentType.String(), i.Arg2Val)
	}
	if i.CommandType == CommandTypePop {
		return fmt.Sprintf("pop %s %d", i.SegmentType.String(), i.Arg2Val)
	}
	return fmt.Sprintf("%s %s %s", i.CommandType.String(), i.Arg1, i.Arg2)
}

func parseInstruction(index int, fileName string, line string) (*Instruction, error) {
	parts := strings.Split(line, " ")
	pl := len(parts)
	if pl == 0 || pl > 3 {
		return nil, fmt.Errorf("invalid instruction length: %d", pl)
	}
	// arithmetic/logical command parsing
	validAL := []string{"add", "sub", "neg", "eq", "gt", "lt", "and", "or", "not"}
	if pl == 1 && slices.Contains(validAL, parts[0]) {
		al := ALTypeAdd
		rawAl := parts[0]
		if rawAl == "add" {
			al = ALTypeAdd
		} else if rawAl == "sub" {
			al = ALTypeSub
		} else if rawAl == "neg" {
			al = ALTypeNeg
		} else if rawAl == "eq" {
			al = ALTypeEq
		} else if rawAl == "gt" {
			al = ALTypeGt
		} else if rawAl == "lt" {
			al = ALTypeLt
		} else if rawAl == "and" {
			al = ALTypeAnd
		} else if rawAl == "or" {
			al = ALTypeOr
		} else if rawAl == "not" {
			al = ALTypeNot
		}
		return &Instruction{
			FileName:    fileName,
			Line:        line,
			CommandType: CommandTypeArithmetic,
			Arg1:        parts[0],
			ALType:      al,
			Index:       index,
		}, nil
	}

	// command type parsing
	ct := CommandTypePush
	rawCt := strings.ToLower(parts[0])
	if rawCt == "pop" {
		ct = CommandTypePop
	} else if rawCt == "push" {
		ct = CommandTypePush
	} else if rawCt == "label" {
		ct = CommandTypeLabel
	} else if rawCt == "goto" {
		ct = CommandTypeGOTO
	} else if rawCt == "if-goto" {
		ct = CommandTypeIf
	} else if rawCt == "function" {
		ct = CommandTypeFunction
	} else if rawCt == "return" {
		ct = CommandTypeReturn
	} else if rawCt == "call" {
		ct = CommandTypeCall
	} else {
		return nil, fmt.Errorf("invalid command type: %s", parts[0])
	}

	// arg1 parsing
	st := SegmentTypeConstant
	arg1 := ""
	if ct == CommandTypePush || ct == CommandTypePop {
		rawSt := strings.ToLower(parts[1])
		arg1 = rawSt
		if rawSt == "constant" {
			st = SegmentTypeConstant
		} else if rawSt == "local" {
			st = SegmentTypeLocal
		} else if rawSt == "argument" {
			st = SegmentTypeArgument
		} else if rawSt == "this" {
			st = SegmentTypeThis
		} else if rawSt == "that" {
			st = SegmentTypeThat
		} else if rawSt == "static" {
			st = SegmentTypeStatic
		} else if rawSt == "temp" {
			st = SegmentTypeTemp
		} else if rawSt == "pointer" {
			st = SegmentTypePointer
		} else {
			return nil, fmt.Errorf("invalid arg1 segment type: %s", parts[1])
		}
	} else if ct == CommandTypeLabel || ct == CommandTypeGOTO || ct == CommandTypeIf || ct == CommandTypeFunction || ct == CommandTypeCall {
		arg1 = parts[1]
	} else if ct == CommandTypeReturn {
		if len(parts) > 1 {
			return nil, fmt.Errorf("invalid arg1 return, no argument expected")
		}
	} else {
		return nil, fmt.Errorf("invalid command type: %s", parts[0])
	}

	// arg2 parsing
	arg2Val := 0
	arg2 := ""
	if ct == CommandTypePush || ct == CommandTypePop ||
		ct == CommandTypeFunction || ct == CommandTypeCall {
		if len(parts) > 2 {
			arg2 = parts[2]
		} else {
			return nil, fmt.Errorf("invalid arg2 value: %s", arg2)
		}
		if len(arg2) > 0 {
			var err error
			arg2Val, err = strconv.Atoi(arg2)
			if err != nil {
				return nil, fmt.Errorf("invalid arg2 value: %s", arg2)
			}
		}
	}

	return &Instruction{
		FileName:    fileName,
		Line:        line,
		CommandType: ct,
		SegmentType: st,
		Arg1:        arg1,
		Arg2:        arg2,
		Arg2Val:     arg2Val,
	}, nil
}

func (i *Instruction) GenAsm() ([]string, error) {
	lines := []string{
		fmt.Sprintf("// %s", i.Line),
	}
	switch i.CommandType {
	case CommandTypeArithmetic:
		aLines, err := i.genArithmetic()
		if err != nil {
			return nil, err
		}
		for _, line := range aLines {
			lines = append(lines, line)
		}
		return lines, nil
	case CommandTypePush:
		switch i.SegmentType {
		case SegmentTypeConstant:
			lines = append(lines, i.genConstantPUSH(i.Arg2Val)...)
		case SegmentTypeStatic:
			lines = append(lines, i.genStaticPUSH()...)
		case SegmentTypeTemp:
			lines = append(lines, i.genTempPUSH()...)
		case SegmentTypePointer:
			lines = append(lines, i.genPointerPUSH()...)
		default:
			lines = append(lines, i.genSegmentPUSH(i.SegmentType, i.Arg2Val)...)
		}
		return lines, nil
	case CommandTypePop:
		switch i.SegmentType {
		case SegmentTypeConstant:
			lines = append(lines, i.genConstantPOP()...)
		case SegmentTypeStatic:
			lines = append(lines, i.genStaticPOP()...)
		case SegmentTypeTemp:
			lines = append(lines, i.genTempPOP()...)
		case SegmentTypePointer:
			lines = append(lines, i.genPointerPOP()...)
		default:
			lines = append(lines, i.genSegmentPOP(i.SegmentType, i.Arg2Val)...)
		}
		return lines, nil
	case CommandTypeLabel:
		lines = append(lines, fmt.Sprintf("(%s)", i.Arg1))
		return lines, nil
	case CommandTypeGOTO:
		lines = append(lines, fmt.Sprintf("@%s", i.Arg1))
		lines = append(lines, "0;JMP")
		return lines, nil
	case CommandTypeIf:
		lines = append(lines, "@SP")
		lines = append(lines, "AM=M-1") // pop & set A to SP-1
		lines = append(lines, "D=M")    // D = value at SP-1
		lines = append(lines, fmt.Sprintf("@%s", i.Arg1))
		lines = append(lines, "D;JNE") // if D != 0, jump to label
		return lines, nil
	case CommandTypeFunction:
		lines = append(lines, i.genFunction()...)
		return lines, nil
	case CommandTypeReturn:
		lines = append(lines, i.genReturn()...)
		return lines, nil
	case CommandTypeCall:
		lines = append(lines, genCall(i.Arg1, i.Arg2Val)...)
		return lines, nil
	}
	return nil, fmt.Errorf("invalid or not handled command with type: %s", i.CommandType.String())
}

func (i *Instruction) genArithmetic() ([]string, error) {
	lines := []string{}
	switch i.ALType {
	case ALTypeAdd, ALTypeSub, ALTypeAnd, ALTypeOr:
		op := ""
		if i.ALType == ALTypeAdd {
			op = "M=D+M"
		} else if i.ALType == ALTypeSub {
			op = "M=M-D"
		} else if i.ALType == ALTypeAnd {
			op = "M=D&M"
		} else if i.ALType == ALTypeOr {
			op = "M=D|M"
		}
		lines = append(lines, "@SP")
		lines = append(lines, "AM=M-1")
		lines = append(lines, "D=M")
		lines = append(lines, "A=A-1")
		lines = append(lines, op)

	case ALTypeNeg:
		lines = append(lines, "@0")
		lines = append(lines, "D=A")
		lines = append(lines, "@SP")
		lines = append(lines, "A=M-1")
		lines = append(lines, "M=D-M")

	case ALTypeEq, ALTypeGt, ALTypeLt:
		id := i.ALType.String()
		lines = append(lines, "@SP")
		lines = append(lines, "AM=M-1")
		lines = append(lines, "D=M")
		lines = append(lines, "A=A-1")
		lines = append(lines, "D=M-D")
		lines = append(lines, i.getLogicalARegister(id+"_true"))
		if i.ALType == ALTypeEq {
			lines = append(lines, "D;JEQ")
		} else if i.ALType == ALTypeGt {
			lines = append(lines, "D;JGT")
		} else if i.ALType == ALTypeLt {
			lines = append(lines, "D;JLT")
		}
		lines = append(lines, "@SP")
		lines = append(lines, "A=M-1")
		lines = append(lines, "M=0") // set to 0 if false
		lines = append(lines, i.getLogicalARegister(id+"_false"))
		lines = append(lines, "0;JMP")

		lines = append(lines, i.getLogicalLabel(id+"_true")) // LABEL
		lines = append(lines, "@SP")
		lines = append(lines, "A=M-1")
		lines = append(lines, "M=-1") // set to -1 if true

		lines = append(lines, i.getLogicalLabel(id+"_false")) // LABEL

	case ALTypeNot:
		lines = append(lines, "@SP")
		lines = append(lines, "A=M-1")
		lines = append(lines, "M=!M")

	default:
		return nil, fmt.Errorf("invalid arithmetic/logical command: %s", i.ALType.String())
	}
	return lines, nil
}

func (i *Instruction) getLogicalLabel(prefix string) string {
	v := fmt.Sprintf("(%s.%d)", prefix, i.Index)
	return strings.ToUpper(v)
}

func (i *Instruction) getLogicalARegister(prefix string) string {
	v := fmt.Sprintf("@%s.%d", prefix, i.Index)
	return strings.ToUpper(v)
}

func (i *Instruction) genConstantPUSH(val int) []string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", val))
	lines = append(lines, "D=A")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	return lines
}

func (i *Instruction) genSegmentPUSH(sgt SegmentType, val int) []string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", val))
	lines = append(lines, "D=A")
	lines = append(lines, fmt.Sprintf("@%s", sgt.ID()))
	lines = append(lines, "A=D+M")
	lines = append(lines, "D=M")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	return lines
}

func (i *Instruction) genStaticPUSH() []string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%s.%d", i.FileName, i.Arg2Val))
	lines = append(lines, "D=M")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	return lines
}

func (i *Instruction) genTempPUSH() []string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", i.Arg2Val)) // offset
	lines = append(lines, "D=A")
	lines = append(lines, "@5")
	lines = append(lines, "A=D+A")
	lines = append(lines, "D=M")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	return lines
}

func (i *Instruction) genPointerPUSH() []string {
	lines := []string{}
	if i.Arg2Val == 0 {
		lines = append(lines, "@THIS")
	} else {
		lines = append(lines, "@THAT")
	}
	lines = append(lines, "D=M")
	lines = append(lines, "@SP")
	lines = append(lines, "A=M")
	lines = append(lines, "M=D")
	lines = append(lines, "@SP")
	lines = append(lines, "M=M+1")
	return lines
}

func (i *Instruction) genConstantPOP() []string {
	lines := []string{}
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	return lines
}

func (i *Instruction) genSegmentPOP(sgt SegmentType, val int) []string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", val))
	lines = append(lines, "D=A")
	lines = append(lines, fmt.Sprintf("@%s", sgt.ID()))
	lines = append(lines, "D=D+M")
	lines = append(lines, "@R13")
	lines = append(lines, "M=D")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	lines = append(lines, "@R13")
	lines = append(lines, "A=M")
	lines = append(lines, "M=D")
	return lines
}

func (i *Instruction) genStaticPOP() []string {
	lines := []string{}
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	lines = append(lines, fmt.Sprintf("@%s.%d", i.FileName, i.Arg2Val))
	lines = append(lines, "M=D")
	return lines
}

func (i *Instruction) genTempPOP() []string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", i.Arg2Val))
	lines = append(lines, "D=A")
	lines = append(lines, "@5")
	lines = append(lines, "D=D+A")
	lines = append(lines, "@R13")
	lines = append(lines, "M=D")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	lines = append(lines, "@R13")
	lines = append(lines, "A=M")
	lines = append(lines, "M=D")
	return lines
}

func (i *Instruction) genPointerPOP() []string {
	lines := []string{}
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	if i.Arg2Val == 0 {
		lines = append(lines, "@THIS")
	} else {
		lines = append(lines, "@THAT")
	}
	lines = append(lines, "M=D")
	return lines
}

func genCall(calleeFn string, calleeNArgs int) []string {
	lines := []string{}

	// push return address
	retAddrLabel := fmt.Sprintf("%s$ret.%d", currentFunctionName, retIndex)
	retIndex++
	lines = append(lines, fmt.Sprintf("/// call ; working with return address %s", retAddrLabel))
	lines = append(lines, "@"+retAddrLabel)
	lines = append(lines, "D=A")
	lines = append(lines, "@SP")
	lines = append(lines, "A=M")
	lines = append(lines, "M=D") // Push return label into the stack
	lines = append(lines, "@SP")
	lines = append(lines, "M=M+1") // inc. SP

	segments := []SegmentType{SegmentTypeLocal, SegmentTypeArgument, SegmentTypeThis, SegmentTypeThat}
	for _, seg := range segments {
		lines = append(lines, fmt.Sprintf("/// call ; working with %s", seg.ID()))
		lines = append(lines, "@"+seg.ID())
		// we had issues here with A=M
		lines = append(lines, "D=M") // segment pointer value
		lines = append(lines, "@SP")
		lines = append(lines, "A=M")
		lines = append(lines, "M=D") // Push segment into the stack
		lines = append(lines, "@SP")
		lines = append(lines, "M=M+1") // inc. SP
	}

	lines = append(lines, "/// call ; ARG = SP - 5 - nArgs")
	lines = append(lines, fmt.Sprintf("@%d", 5+calleeNArgs))
	lines = append(lines, "D=A")
	lines = append(lines, "@SP")
	lines = append(lines, "A=M")
	lines = append(lines, "D=A-D")
	lines = append(lines, "@ARG")
	lines = append(lines, "M=D") // ARG = SP - 5 - nArgs

	lines = append(lines, "/// call ; LCL = SP")
	lines = append(lines, "@SP")
	// we had issues here with A=M
	lines = append(lines, "D=M")
	lines = append(lines, "@LCL")
	lines = append(lines, "M=D") // LCL = SP

	lines = append(lines, "/// call ; goto function "+calleeFn)
	lines = append(lines, "@"+calleeFn)
	lines = append(lines, "0;JMP")

	lines = append(lines, fmt.Sprintf("(%s)", retAddrLabel))

	return lines
}

// Handling: function functionName nVars
func (i *Instruction) genFunction() []string {
	lines := []string{}
	currentFunctionName = i.Arg1

	lines = append(lines, fmt.Sprintf("(%s)", i.Arg1))
	for range i.Arg2Val {
		lines = append(lines, i.genConstantPUSH(0)...)
	}
	return lines
}

// Handling: return
func (i *Instruction) genReturn() []string {
	lines := []string{}
	lines = append(lines, "/// return ; endFrame = LCL")
	lines = append(lines, "@LCL")
	lines = append(lines, "D=M")
	lines = append(lines, "@R13")
	lines = append(lines, "M=D") /// endFrame = LCL ///

	lines = append(lines, "/// return ; retAddr = D = RAM[endFrame - 5]")
	lines = append(lines, "@5")
	lines = append(lines, "A=D-A") // endFrame - 5
	lines = append(lines, "D=M")   // D = RAM[endFrame - 5]
	lines = append(lines, "@R14")
	lines = append(lines, "M=D") /// retAddr = D = RAM[endFrame - 5] ///

	lines = append(lines, "/// return ; RAM[ARG] = pop() = RAM[SP-1]")
	lines = append(lines, "@SP")
	lines = append(lines, "A=M-1") // A = SP - 1
	lines = append(lines, "D=M")   // D = RAM[SP - 1] = return value
	lines = append(lines, "@ARG")
	lines = append(lines, "A=M")
	lines = append(lines, "M=D") // RAM[ARG] = pop() = RAM[SP-1]

	lines = append(lines, "/// return ; SP = ARG + 1")
	lines = append(lines, "@ARG")
	lines = append(lines, "D=M+1")
	lines = append(lines, "@SP")
	lines = append(lines, "M=D") // SP = ARG + 1

	segments := []SegmentType{SegmentTypeThat, SegmentTypeThis, SegmentTypeArgument, SegmentTypeLocal}
	for _, seg := range segments {
		lines = append(lines, fmt.Sprintf("/// %s ; working with %s", i.Line, seg.String()))
		lines = append(lines, "@R13")
		lines = append(lines, "ADM=M-1")
		lines = append(lines, "D=M")
		lines = append(lines, "@"+seg.ID())
		lines = append(lines, "M=D") // seg = *(endFrame – 1)
	}

	lines = append(lines, "/// return ; goto caller")
	lines = append(lines, "@R14")
	lines = append(lines, "A=M")
	lines = append(lines, "0;JMP") // goto retAddr
	return lines
}
