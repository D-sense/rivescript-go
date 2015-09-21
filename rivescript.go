/*
Package rivescript implements the RiveScript chatbot scripting language.
*/
package rivescript

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	//_Parser "./parser.go"
)

// Constants
const RS_VERSION float64 = 2.0

/******************************************************************************
 * Constructor and Debug Methods                                              *
 ******************************************************************************/

type RiveScript struct {
	// Parameters
	Debug  bool // Debug mode
	Strict bool // Strictly enforce RiveScript syntax
	Depth  int  // Max depth for recursion
	UTF8   bool // Support UTF-8 RiveScript code
	UnicodePunctuation *regexp.Regexp

	// Internal data structures
	global   map[string]string          // 'global' variables
	var_     map[string]string          // 'var' bot variables
	sub      map[string]string          // 'sub' substitutions
	person   map[string]string          // 'person' substitutions
	array    map[string][]string        // 'array'
	users    map[string]*UserData       // user variables
	freeze   map[string][]*UserData     // frozen user variables
	includes map[string]map[string]bool // included topics
	inherits map[string]map[string]bool // inherited topics
	objlangs map[string]string          // object macro languages]
	topics   map[string]*astTopic       // main topic structure
	thats    map[string]*thatTopic      // %Previous mapper
	sorted   *sortBuffer                // Sorted data from SortReplies()

	// State information.
	currentUser string
}

func New() *RiveScript {
	rs := new(RiveScript)
	rs.Debug = false
	rs.Strict = true
	rs.Depth = 50
	rs.UTF8 = false
	rs.UnicodePunctuation = regexp.MustCompile(`[.,!?;:]`)

	// Initialize all the data structures.
	rs.global = map[string]string{}
	rs.var_ = map[string]string{}
	rs.sub = map[string]string{}
	rs.person = map[string]string{}
	rs.array = map[string][]string{}
	rs.users = map[string]*UserData{}
	//rs.freeze
	rs.includes = map[string]map[string]bool{}
	rs.inherits = map[string]map[string]bool{}
	rs.topics = map[string]*astTopic{}
	rs.thats = map[string]*thatTopic{}
	rs.sorted = new(sortBuffer)
	return rs
}

func (rs RiveScript) Version() string {
	// TODO: versioning
	return "0.0.1"
}

// say prints a debugging message
func (rs RiveScript) say(message string, a ...interface{}) {
	if rs.Debug {
		fmt.Printf(message+"\n", a...)
	}
}

// warn prints a warning message for non-fatal errors
func (rs RiveScript) warn(message string, a ...interface{}) {
	fmt.Printf("[WARN] "+message+"\n", a...)
}

// syntax is like warn but takes a filename and line number.
func (rs RiveScript) warnSyntax(message string, filename string, lineno int, a ...interface{}) {
	message += fmt.Sprintf(" at %s line %d", filename, lineno)
	rs.warn(message, a...)
}

/******************************************************************************
 * Loading methods                                                            *
 ******************************************************************************/

/*
LoadFile loads a single RiveScript source file from disk.

Params:
- path: File path to
*/
func (rs RiveScript) LoadFile(path string) {
	rs.say("Load RiveScript file: %s", path)

	fh, err := os.Open(path)
	if err != nil {
		rs.warn("Failed to open file %s: %s", path, err)
		return
	}

	defer fh.Close()
	scanner := bufio.NewScanner(fh)
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	rs.parse(path, lines)
}

/*
LoadDirectory loads multiple RiveScript documents from a folder on disk.

Params:
- path: Path to the directory on disk
- extensions...: List of file extensions to filter on, default is '.rive' and '.rs'
*/
func (rs RiveScript) LoadDirectory(path string, extensions ...string) {
	if len(extensions) == 0 {
		extensions = []string{".rive", ".rs"}
	}

	files, err := filepath.Glob(fmt.Sprintf("%s/*", path))
	if err != nil {
		rs.warn("Failed to open folder %s: %s", path, err)
		return
	}

	for _, f := range files {
		rs.LoadFile(f)
	}
}

/*
Stream loads RiveScript code from a text buffer.

Params:
- code: Raw source code of a RiveScript document, with line breaks after each line.
*/
func (rs RiveScript) Stream(code string) {
	lines := strings.Split(code, "\n")
	rs.parse("Stream()", lines)
}

/*
parse loads the RiveScript code into the bot's memory.
*/
func (rs RiveScript) parse(path string, lines []string) {
	rs.say("Parsing code!")

	// Get the "abstract syntax tree" of this file.
	ast := rs.ParseSource(path, lines)

	// Get all of the "begin" type variables
	for k, v := range ast.begin.global {
		if v == "<undef>" {
			delete(rs.global, k)
		} else {
			rs.global[k] = v
		}
	}
	for k, v := range ast.begin.var_ {
		if v == "<undef>" {
			delete(rs.var_, k)
		} else {
			rs.var_[k] = v
		}
	}
	for k, v := range ast.begin.sub {
		if v == "<undef>" {
			delete(rs.sub, k)
		} else {
			rs.sub[k] = v
		}
	}
	for k, v := range ast.begin.person {
		if v == "<undef>" {
			delete(rs.person, k)
		} else {
			rs.person[k] = v
		}
	}
	for k, v := range ast.begin.array {
		rs.array[k] = v
	}

	// Consume all the parsed triggers.
	for topic, data := range ast.topics {
		// Keep a map of the topics that are included/inherited under this topic.
		if _, ok := rs.includes[topic]; !ok {
			rs.includes[topic] = map[string]bool{}
		}
		if _, ok := rs.inherits[topic]; !ok {
			rs.inherits[topic] = map[string]bool{}
		}

		// Merge in the topic inclusions/inherits.
		for included, _ := range data.includes {
			rs.includes[topic][included] = true
		}
		for inherited, _ := range data.inherits {
			rs.inherits[topic][inherited] = true
		}

		// Initialize the topic structure.
		if _, ok := rs.topics[topic]; !ok {
			rs.topics[topic] = new(astTopic)
			rs.topics[topic].triggers = []*astTrigger{}
		}

		// Consume the AST triggers into the brain.
		for _, trigger := range data.triggers {
			rs.topics[topic].triggers = append(rs.topics[topic].triggers, trigger)

			// Does this one have a %Previous? If so, make a pointer to this
			// exact trigger in rs.thats
			if trigger.previous != "" {
				// Initialize the structure first.
				if _, ok := rs.thats[topic]; !ok {
					rs.thats[topic] = new(thatTopic)
					rs.say("%q", rs.thats[topic])
					rs.thats[topic].triggers = map[string]*thatTrigger{}
				}
				if _, ok := rs.thats[topic].triggers[trigger.trigger]; !ok {
					rs.say("%q", rs.thats[topic].triggers[trigger.trigger])
					rs.thats[topic].triggers[trigger.trigger] = new(thatTrigger)
					rs.thats[topic].triggers[trigger.trigger].previous = map[string]*astTrigger{}
				}
				rs.thats[topic].triggers[trigger.trigger].previous[trigger.previous] = trigger
			}
		}
	}
}

/*
SortReplies sorts the reply structures in memory for optimal matching.

After you have finished loading your RiveScript code, call this method to
populate the various sort buffers. This is absolutely necessary for reply
matching to work efficiently!
*/
func (rs RiveScript) SortReplies() {
	// (Re)initialize the sort cache.
	rs.sorted.topics = map[string][]sortedTriggerEntry{}
	rs.sorted.thats = map[string][]sortedTriggerEntry{}
	rs.say("Sorting triggers...")

	// Loop through all the topics.
	for topic, _ := range rs.topics {
		rs.say("Analyzing topic %s", topic)

		// Collect a list of all the triggers we're going to worry about. If this
		// topic inherits another topic, we need to recursively add those to the
		// list as well.
		allTriggers := rs.getTopicTriggers(topic, rs.topics, nil)

		// Sort these triggers.
		rs.sorted.topics[topic] = rs.sortTriggerSet(allTriggers, true)

		// Get all of the %Previous triggers for this topic.
		thatTriggers := rs.getTopicTriggers(topic, nil, rs.thats)

		// And sort them, too.
		rs.sorted.thats[topic] = rs.sortTriggerSet(thatTriggers, false)
	}

	// Sort the substitution lists.
	rs.sorted.sub = sortList(rs.sub)
	rs.sorted.person = sortList(rs.person)
}

// DumpTopics is a debug method which dumps the topic structure from the bot's memory.
func (rs RiveScript) DumpTopics() {
	for topic, data := range rs.topics {
		fmt.Printf("Topic: %s\n", topic)
		for _, trigger := range data.triggers {
			fmt.Printf("  + %s\n", trigger.trigger)
			if trigger.previous != "" {
				fmt.Printf("    %% %s\n", trigger.previous)
			}
			for _, cond := range trigger.condition {
				fmt.Printf("    * %s\n", cond)
			}
			for _, reply := range trigger.reply {
				fmt.Printf("    - %s\n", reply)
			}
			if trigger.redirect != "" {
				fmt.Printf("    @ %s\n", trigger.redirect)
			}
		}
	}
}

// DumpSorted is a debug method which dumps the sort tree from the bot's memory.
func (rs RiveScript) DumpSorted() {
	rs._dumpSorted(rs.sorted.topics, "Topics")
	rs._dumpSorted(rs.sorted.thats, "Thats")
	rs._dumpSortedList(rs.sorted.sub, "Substitutions")
	rs._dumpSortedList(rs.sorted.person, "Person Substitutions")
}
func (rs RiveScript) _dumpSorted(tree map[string][]sortedTriggerEntry, label string) {
	fmt.Printf("Sort Buffer: %s\n", label)
	for topic, data := range tree {
		fmt.Printf("  Topic: %s\n", topic)
		for _, trigger := range data {
			fmt.Printf("    + %s\n", trigger.trigger)
		}
	}
}
func (rs RiveScript) _dumpSortedList(list []string, label string) {
	fmt.Printf("Sort buffer: %s\n", label)
	for _, item := range list {
		fmt.Printf("  %s\n", item)
	}
}
