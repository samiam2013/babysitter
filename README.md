# BabySitter

Do you have code that you need to test but it's the kind of code that just keeps running at a randomly decided on duration instead of using `cron`?

I'm sorry to hear that. I've been there. I've done that. I've got the t-shirt.

I've also got the solution for you.

## Installation

`go install github.com/samiam2013/babysitter@latest`

## Usage

`babysitter -kill_on 'success' -- <command>`

## Explanation

`-kill_on` allows you to specify a string of text that will be listed for by babysitter. When the code starts, you command is spun off to a new process and babysitter will watch for the string you specified. When it sees that string, it will kill the process and exit with a 0 exit code.

This allows babysitter to simply wrap your non-stopping code so that an integration test can continue on to query the database, change a branch, and run the test again (my original use case for this).