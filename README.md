# `ticktickrules`

Package ticktickrules provides a basic cron-like rule matcher for doing simple calculations of
cron expressions. It exposes functionality for determining the next time a cron expression is matched.

Only the simple cron rules are available but this is pretty much good enough for most applications. If you
want to support things like @hourly, @weekly, etc then you should combine this with higher level time windows.

See the documentation at [godoc.org/github.com/AstromechZA/ticktickrules](https://godoc.org/github.com/AstromechZA/ticktickrules).

### Example:

A simple pretend cron example:

```golang
import "github.com/AstromechZA/ticktickrules"

func Something() {

    // This rule matches the first minute every 2 hours
    rule, err := ticktickrules.NewRule("0", "*/2", "*", "*", "*")
    // There might be an error if we wrote unsupported rules
    if err != nil {
        panic(err.Error())
    }

    // Loop forever
    for {
        // Naive sleeping
        time.Sleep(rule.UntilNext(time.Now()))
        fmt.Println("Hi!)
    }
}

```
