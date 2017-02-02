package rivescript

// Miscellaneous structures

// Forms of undefined.
const (
	// UNDEFINED is the text "undefined", the default text for variable getters.
	UNDEFINED = "undefined"

	// UNDEFTAG is the "<undef>" tag for unsetting variables in !Definitions.
	UNDEFTAG = "<undef>"
)

// Subroutine is a Golang function type for defining an object macro in Go.
// TODO: get this exportable to third party devs somehow
type Subroutine func(*RiveScript, []string) string

// Sort buffer data, for RiveScript.SortReplies()
type sortBuffer struct {
	topics map[string][]sortedTriggerEntry // Topic name -> array of triggers
	thats  map[string][]sortedTriggerEntry
	sub    []string // Substitutions
	person []string // Person substitutions
}

// Holds a sorted trigger and the pointer to that trigger's data
type sortedTriggerEntry struct {
	trigger string
	pointer *astTrigger
}

// Temporary categorization of triggers while sorting
type sortTrack struct {
	atomic map[int][]sortedTriggerEntry // Sort by number of whole words
	option map[int][]sortedTriggerEntry // Sort optionals by number of words
	alpha  map[int][]sortedTriggerEntry // Sort alpha wildcards by no. of words
	number map[int][]sortedTriggerEntry // Sort numeric wildcards by no. of words
	wild   map[int][]sortedTriggerEntry // Sort wildcards by no. of words
	pound  []sortedTriggerEntry         // Triggers of just '#'
	under  []sortedTriggerEntry         // Triggers of just '_'
	star   []sortedTriggerEntry         // Triggers of just '*'
}
