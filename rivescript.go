/*
Package rivescript implements the RiveScript chatbot scripting language.
*/
package rivescript

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	//_Parser "./parser.go"
)

// Constants
const RS_VERSION float64 = 2.0

type RiveScript struct {
	// Parameters
	Debug  bool // Debug mode
	Strict bool // Strictly enforce RiveScript syntax
	Depth  int  // Max depth for recursion
	UTF8   bool // Support UTF-8 RiveScript code

	// Internal data structures
	global   map[string]string      // 'global' variables
	var_     map[string]string      // 'var' bot variables
	sub      map[string]string      // 'sub' substitutions
	person   map[string]string      // 'person' substitutions
	array    map[string][]string    // 'array'
	users    map[string][]*UserData // user variables
	freeze   map[string][]*UserData // frozen user variables
	includes map[string]string      // included topics
	inherits map[string]string      // inherited topics
	objlangs map[string]string      // object macro languages]
	topics   map[string]*astTopic   // main topic structure
	thats    map[string]*thatTopic   // %Previous mapper
}

func New() *RiveScript {
	rs := new(RiveScript)
	rs.Debug = false
	rs.Strict = true
	rs.Depth = 50
	rs.UTF8 = false

	// Initialize all the data structures.
	rs.global = map[string]string{}
	rs.var_ = map[string]string{}
	rs.sub = map[string]string{}
	rs.person = map[string]string{}
	rs.array = map[string][]string{}
	//rs.users = map[string]*UserData{}
	//rs.freeze
	rs.topics = map[string]*astTopic{}
	rs.thats = map[string]*thatTopic{}
	return rs
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
		// if val, ok := rs.includes[topic]; !ok {
		// 	rs.includes[topic] = map[string]bool{}
		// }
		// if val, ok := rs.inherits[topic]; !ok {
		// 	rs.inherits[topic] = map[string]bool{}
		// }
		// TODO: merge these in

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

// say prints a debugging message
func (rs RiveScript) say(message string, a ...interface{}) {
	if rs.Debug {
		fmt.Printf(message+"\n", a...)
	}
}

// warn prints a warning message for non-fatal errors
func (rs RiveScript) warn(message string, a ...interface{}) {
	fmt.Printf(message+"\n", a...)
}

// syntax is like warn but takes a filename and line number.
func (rs RiveScript) warnSyntax(message string, filename string, lineno int, a ...interface{}) {
	message += fmt.Sprintf(" at %s line %d", filename, lineno)
	rs.warn(message, a...)
}
