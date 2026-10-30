package kosac

type Level string

const (
	R Level = "R" // rejects the submission
	W Level = "W" // passes with a warning
)

// Issue is one finding of the template read or a check.
type Issue struct {
	Rule                      string // S05, C01…C18, XSD-DDI, XSD-KOSAC, PLACE, REF
	Level                     Level
	Sheet, Cell               string // where the value was read, when it came from the template
	Path                      string // XML path, only from ValidateXML
	FileNo, VarName, Code     string // machine keys of the row
	Actual, Expected, Message string // Message is one Korean sentence
}
