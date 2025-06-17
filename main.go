package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	var srcFile, dstFile, cmpFile string
	flag.StringVar(&srcFile, "s", "", "source file in vm extension (e.g. Add.vm)")
	flag.StringVar(&dstFile, "o", "", "destination file (e.g. Add.asm)")
	flag.StringVar(&cmpFile, "c", "", "compare file")
	flag.Parse()
	if srcFile == "" {
		fmt.Println("No source file provided")
		flag.Usage()
		os.Exit(1)
	}

	// check if the file is of asm extension
	if filepath.Ext(srcFile) != ".vm" {
		fmt.Println("Source file must be of vm extension")
		os.Exit(1)
	}

	if dstFile == "" {
		dstFile = filepath.Base(srcFile)
		dstFile = strings.TrimSuffix(dstFile, filepath.Ext(dstFile))
		dstFile = dstFile + ".asm"
	}

	srcF, err := os.Open(srcFile)
	if err != nil {
		fmt.Println("Error opening source file", err)
		os.Exit(1)
	}
	defer srcF.Close()

	// create dst if not exists
	var dstF *os.File
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		dstF, err = os.Create(dstFile)
		if err != nil {
			fmt.Println("Error creating destination file", err)
			os.Exit(1)
		}
	} else {
		dstF, err = os.OpenFile(dstFile, os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening destination file", err)
			os.Exit(1)
		}
	}
	defer dstF.Close()

	rawLines := []string{}

	scanner := bufio.NewScanner(srcF)
	for scanner.Scan() {
		line := scanner.Text()
		rawLines = append(rawLines, line)
	}

	instructionsLines := []string{}
	for _, line := range rawLines {
		line = removeCommentsAndSpaces(line)
		if line == "" {
			continue
		}
		instructionsLines = append(instructionsLines, line)
	}

	resultLines := []string{}

	for i, line := range instructionsLines {
		filename := filepath.Base(srcFile)
		filename = strings.TrimSuffix(filename, filepath.Ext(filename))
		instruction, err := parseInstruction(i, filename, line)
		if err != nil {
			fmt.Println("Error parsing instruction", err)
			os.Exit(2)
		}
		asm, err := instruction.GenAsm()
		if err != nil {
			fmt.Println("Error generating asm", err)
			os.Exit(2)
		}
		resultLines = append(resultLines, asm)
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

type CommandType int
type SegmentType int
type ALType int

const (
	CommandTypeArithmetic CommandType = iota
	CommandTypePush
	CommandTypePop
)

func (ct CommandType) String() string {
	return []string{
		"arithmetic",
		"push",
		"pop",
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

type Instruction struct {
	FileName    string
	Line        string
	CommandType CommandType
	Arg1        string
	SegmentType SegmentType
	Arg2        string
	valI        int
	ALType      ALType
	Index       int
}

func (i *Instruction) String() string {
	if i.CommandType == CommandTypeArithmetic {
		return i.Arg1
	}
	if i.CommandType == CommandTypePush {
		return fmt.Sprintf("push %s %d", i.SegmentType.String(), i.valI)
	}
	if i.CommandType == CommandTypePop {
		return fmt.Sprintf("pop %s %d", i.SegmentType.String(), i.valI)
	}
	return fmt.Sprintf("%s %s %s", i.CommandType.String(), i.Arg1, i.Arg2)
}

func parseInstruction(index int, fileName string, line string) (*Instruction, error) {
	parts := strings.Split(line, " ")
	pl := len(parts)
	if pl == 0 || pl > 3 {
		return nil, fmt.Errorf("invalid instruction length: %d", pl)
	}
	if pl == 1 {
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
	ct := CommandTypePush
	rawCt := parts[0]
	if rawCt == "pop" {
		ct = CommandTypePop
	} else if rawCt == "push" {
		ct = CommandTypePush
	} else {
		return nil, fmt.Errorf("invalid command type: %s", rawCt)
	}
	st := SegmentTypeConstant
	rawSt := parts[1]
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
		return nil, fmt.Errorf("invalid segment type: %s", rawSt)
	}
	valI := 0
	rawValI := parts[2]
	if len(rawValI) > 0 {
		var err error
		valI, err = strconv.Atoi(rawValI)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", rawValI)
		}
	}
	return &Instruction{
		FileName:    fileName,
		Line:        line,
		CommandType: ct,
		SegmentType: st,
		Arg1:        rawSt,
		Arg2:        rawValI,
		valI:        valI,
	}, nil
}

func (i *Instruction) joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

func (i *Instruction) GenAsm() (string, error) {
	lines := []string{
		fmt.Sprintf("// %s", i.Line),
	}
	if i.CommandType == CommandTypeArithmetic {
		aLines, err := i.genArithmetic()
		if err != nil {
			return "", err
		}
		for _, line := range aLines {
			lines = append(lines, line)
		}
		return i.joinLines(lines), nil
	}
	if i.CommandType == CommandTypePush {
		switch i.SegmentType {
		case SegmentTypeConstant:
			lines = append(lines, i.genConstantPUSH())
		case SegmentTypeStatic:
			lines = append(lines, i.genStaticPUSH())
		case SegmentTypeTemp:
			lines = append(lines, i.genTempPUSH())
		case SegmentTypePointer:
			lines = append(lines, i.genPointerPUSH())
		default:
			lines = append(lines, i.genSegmentPUSH())
		}
		return i.joinLines(lines), nil
	}
	if i.CommandType == CommandTypePop {
		switch i.SegmentType {
		case SegmentTypeConstant:
			lines = append(lines, i.genConstantPOP())
		case SegmentTypeStatic:
			lines = append(lines, i.genStaticPOP())
		case SegmentTypeTemp:
			lines = append(lines, i.genTempPOP())
		case SegmentTypePointer:
			lines = append(lines, i.genPointerPOP())
		default:
			lines = append(lines, i.genSegmentPOP())
		}
		return i.joinLines(lines), nil
	}
	return "", fmt.Errorf("invalid command type: %s", i.CommandType.String())
}

func (i *Instruction) genConstantPUSH() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", i.valI))
	lines = append(lines, "D=A")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	// lines = append(lines, "@SP")
	// lines = append(lines, "A=M")
	// lines = append(lines, "M=D")
	// lines = append(lines, "@SP")
	// lines = append(lines, "M=M+1")
	return i.joinLines(lines)
}

func (i *Instruction) genSegmentPUSH() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", i.valI))
	lines = append(lines, "D=A")
	lines = append(lines, fmt.Sprintf("@%s", i.SegmentType.ID()))
	lines = append(lines, "A=D+M")
	lines = append(lines, "D=M")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	return i.joinLines(lines)
}

func (i *Instruction) genStaticPUSH() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%s.%d", i.FileName, i.valI))
	lines = append(lines, "D=M")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	return i.joinLines(lines)
}

func (i *Instruction) genTempPUSH() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", i.valI)) // offset
	lines = append(lines, "D=A")
	lines = append(lines, "@5")
	lines = append(lines, "A=D+A")
	lines = append(lines, "D=M")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M+1")
	lines = append(lines, "A=A-1")
	lines = append(lines, "M=D")
	return i.joinLines(lines)
}

func (i *Instruction) genPointerPUSH() string {
	lines := []string{}
	if i.valI == 0 {
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
	return i.joinLines(lines)
}

func (i *Instruction) genConstantPOP() string {
	lines := []string{}
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	return i.joinLines(lines)
}

func (i *Instruction) genSegmentPOP() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", i.valI))
	lines = append(lines, "D=A")
	lines = append(lines, fmt.Sprintf("@%s", i.SegmentType.ID()))
	lines = append(lines, "D=D+M")
	lines = append(lines, "@R13")
	lines = append(lines, "M=D")
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	lines = append(lines, "@R13")
	lines = append(lines, "A=M")
	lines = append(lines, "M=D")
	return i.joinLines(lines)
}

func (i *Instruction) genStaticPOP() string {
	lines := []string{}
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	lines = append(lines, fmt.Sprintf("@%s.%d", i.FileName, i.valI))
	lines = append(lines, "M=D")
	return i.joinLines(lines)
}

func (i *Instruction) genTempPOP() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("@%d", i.valI))
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
	return i.joinLines(lines)
}

func (i *Instruction) genPointerPOP() string {
	lines := []string{}
	lines = append(lines, "@SP")
	lines = append(lines, "AM=M-1")
	lines = append(lines, "D=M")
	if i.valI == 0 {
		lines = append(lines, "@THIS")
	} else {
		lines = append(lines, "@THAT")
	}
	lines = append(lines, "M=D")
	return i.joinLines(lines)
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
