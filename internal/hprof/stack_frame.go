package hprof

import (
	"fmt"
	"strings"
)

type ThreadStack struct {
	ThreadID    int32
	StackFrames []StackFrame
	IsAlive     bool
}

func BuildThreadStacks(stackTraces []StackTrace, stackFrames []StackFrame, threadStatus map[int32]bool) ([]ThreadStack, error) {
	var threadStacks []ThreadStack

	for _, trace := range stackTraces {
		var frames []StackFrame
		for _, frameID := range trace.FramesID {
			if frameID >= 0 && int(frameID) < len(stackFrames) {
				frames = append(frames, stackFrames[frameID])
			}
		}

		threadStacks = append(threadStacks, ThreadStack{
			ThreadID:    trace.ThreadSerialNumber,
			StackFrames: frames,
			IsAlive:     threadStatus[trace.ThreadSerialNumber],
		})
	}

	return threadStacks, nil
}

func convertSignature(signature string) string {
	typeMap := map[rune]string{
		'V': "void",
		'Z': "boolean",
		'B': "byte",
		'S': "short",
		'I': "int",
		'J': "long",
		'F': "float",
		'D': "double",
		'C': "char",
		'[': "[]",
	}

	var buffer strings.Builder
	reader := strings.NewReader(signature)
	paramsMode := true
	arrayDepth := 0

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		switch {
		case r == '(':
			buffer.WriteString("(")
		case r == ')':
			paramsMode = false
			if buffer.Len() > 0 && buffer.String()[buffer.Len()-1] == ',' {
				current := buffer.String()
				buffer.Reset()
				buffer.WriteString(current[:len(current)-2])
			}
			buffer.WriteString(") => ")
		case paramsMode:
			if r == '[' {
				arrayDepth++
				continue
			}

			if r == 'L' {
				var className strings.Builder
				for {
					cr, _, err := reader.ReadRune()
					if err != nil || cr == ';' {
						break
					}
					className.WriteRune(cr)
				}
				classNameStr := strings.Replace(className.String(), "/", ".", -1)
				buffer.WriteString(classNameStr)
				for i := 0; i < arrayDepth; i++ {
					buffer.WriteString("[]")
				}
				arrayDepth = 0
				buffer.WriteString(", ")
			} else if typeName, ok := typeMap[r]; ok {
				buffer.WriteString(typeName)
				for i := 0; i < arrayDepth; i++ {
					buffer.WriteString("[]")
				}
				arrayDepth = 0
				buffer.WriteString(", ")
			}
		default:
			if r == 'L' {
				var className strings.Builder
				for {
					cr, _, err := reader.ReadRune()
					if err != nil || cr == ';' {
						break
					}
					className.WriteRune(cr)
				}
				classNameStr := strings.Replace(className.String(), "/", ".", -1)
				buffer.WriteString(classNameStr)
			} else if typeName, ok := typeMap[r]; ok {
				buffer.WriteString(typeName)
			}
		}
	}

	result := buffer.String()
	if strings.HasPrefix(result, "()") {
		result = strings.Replace(result, "(), ", "()", 1)
	}
	return result
}
func PrintStackInfo(traces []StackTrace, frames []StackFrame, stacks []ThreadStack, idMap map[int64]string) {
	printStackTraces(traces, idMap)
	printStackFrames(frames, idMap)
	printThreadStacks(stacks, idMap)
}

func printStackTraces(traces []StackTrace, idMap map[int64]string) {
	fmt.Println("\n--- Stack Traces ---")
	for _, trace := range traces {
		threadName := "unknown"
		if name, ok := idMap[int64(trace.ThreadSerialNumber)]; ok {
			threadName = name
		}

		fmt.Printf("[Trace #%d] Thread: %s (%d)\n",
			trace.StackTraceSerialNumber,
			threadName,
			trace.ThreadSerialNumber,
		)
	}
}

func printStackFrames(frames []StackFrame, idMap map[int64]string) {
	fmt.Println("\n--- Stack Frames ---")
	for i, frame := range frames {
		signature := "unknown"
		if sig, ok := idMap[frame.MethodSignatureStringId]; ok {
			signature = convertSignature(sig)
		}

		sourceFile := "unknown"
		if file, ok := idMap[frame.SourceFileNameStringId]; ok {
			sourceFile = file
		}

		fmt.Printf("Frame #%d:\nMethod ID: %d\nSignature: %s\nSource: %s\n\n",
			i+1,
			frame.MethodId,
			signature,
			sourceFile,
		)
	}
}

func printThreadStacks(stacks []ThreadStack, idMap map[int64]string) {
	fmt.Println("\n--- Thread Stacks ---")
	for _, stack := range stacks {
		status := "DEAD"
		if stack.IsAlive {
			status = "ALIVE"
		}

		fmt.Printf("\n[Thread %d - %s]\n", stack.ThreadID, status)

		if len(stack.StackFrames) == 0 {
			fmt.Println("  No stack frames")
			continue
		}

		for i, frame := range stack.StackFrames {
			methodName := "unknown"
			if name, ok := idMap[frame.MethodId]; ok {
				methodName = name
			}

			signature := "unknown"
			if sig, ok := idMap[frame.MethodSignatureStringId]; ok {
				signature = convertSignature(sig)
			}

			fmt.Printf("  %d. %s %s\n", i+1, methodName, signature)
		}
	}
}
