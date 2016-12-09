package rivescript_test

import (
	"fmt"

	"github.com/aichaos/rivescript-go"
	"github.com/aichaos/rivescript-go/config"
	"github.com/aichaos/rivescript-go/lang/javascript"
	rss "github.com/aichaos/rivescript-go/src"
)

func ExampleRiveScript() {
	bot := rivescript.New(config.Basic())

	// Load a directory full of RiveScript documents (.rive files)
	bot.LoadDirectory("eg/brain")

	// Load an individual file.
	bot.LoadFile("testsuite.rive")

	// Sort the replies after loading them!
	bot.SortReplies()

	// Get a reply.
	reply := bot.Reply("local-user", "Hello, bot!")
	fmt.Printf("The bot says: %s", reply)
}

func ExampleRiveScript_javascript() {
	// Example for configuring the JavaScript object macro handler via Otto.

	bot := rivescript.New(config.Basic())

	// Create the JS handler.
	bot.SetHandler("javascript", javascript.New(bot))

	// Now we can use object macros written in JS!
	bot.Stream(`
		> object add javascript
			var a = args[0];
			var b = args[1];
			return parseInt(a) + parseInt(b);
		< object

		> object setname javascript
			// Set the user's name via JavaScript
			var uid = rs.CurrentUser();
			rs.SetUservar(uid, args[0], args[1])
		< object

		+ add # and #
		- <star1> + <star2> = <call>add <star1> <star2></call>

		+ my name is *
		- I will remember that.<call>setname <id> <formal></call>

		+ what is my name
		- You are <get name>.
	`)
	bot.SortReplies()

	reply := bot.Reply("local-user", "Add 5 and 7")
	fmt.Printf("Bot: %s\n", reply)
}

func ExampleRiveScript_subroutine() {
	// Example for defining a Go function as an object macro.
	// import rss "github.com/aichaos/rivescript-go/src"

	bot := rivescript.New(config.Basic())

	// Define an object macro named `setname`
	bot.SetSubroutine("setname", func(rs *rss.RiveScript, args []string) string {
		uid := rs.CurrentUser()
		rs.SetUservar(uid, args[0], args[1])
		return ""
	})

	// Stream in some RiveScript code.
	bot.Stream(`
		+ my name is *
		- I will remember that.<call>setname <id> <formal></call>

		+ what is my name
		- You are <get name>.
	`)
	bot.SortReplies()

	_ = bot.Reply("local-user", "my name is bob")
	reply := bot.Reply("local-user", "What is my name?")
	fmt.Printf("Bot: %s\n", reply)
}
