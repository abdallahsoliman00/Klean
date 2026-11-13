# Intro
This is the backend-specific Agents file. These are rules to follow when writing the backend.

# Code cleanliness
The code must contain the least number of magic values. if you are to add any constants, use the files in ./internal/config/ to store these values or create a new file at that location.

The code you write must be modular. If there is code that can be reused, then reuse it, and always remember the DRY principle: Don't Repeat Yourself.
If you are at a crossroads between writing a shorter, more complicated and a longer less complicated function, choose the less complicated version. always go for the least amount of complexity.

Never write any tests for functions that are placeholders or genertate test values. Only write tests for functions that are actually used as is in the code.

# Function Honesty
There are two types of functions: honest functions and dishonest functions.
Honest functions are those that are completely predicatable and stateless.
Take this example:
```go
func add(a int, b int) int (
    return a + b
)
```
This is an honest function because adding the same two numbers always yields the same output. This type of function is testable and can be easily reasoned about.

Dishonest functions are those that depend on the system state such as `get_time()` or `get_user_input()`. These depend entirely on system state or external factors.

The issue with dishonest funcions is that they are infectious. If a dishonest function is called within an honest function, this hoest function then becomes dishonest. So, to minimize unpredictability and keep as much of the system honest, keep dishonest functions at the highest level of function calls possible.
