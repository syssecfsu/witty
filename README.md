# Web Terminal
A (unsafe) technical demo to export a shell to web browser. 

This program is written in the go programming language, using the 
Gin web framework, gorilla/websocket, pty, and xterm.js!
The workflow is simple, the client will initiate a terminal 
window (xterm.js) and create a websocket with the server. On 
the server side, it serves the basic HTML/JS/CSS files and 
websockets (by shovling the data between pty and xterm).

It is amazing what you can do with 200 lines of go code. 

It is just a simple demo in case some people are interested in 
how to setup xterm.js with websocket. 

To use the program, download/clone the code, and in the web_terminal
directory, run ```go build .```, this will create the binary called
web_terminal. To run it, use ```./web_terminal cmd options_to_cmd```.
If no cmd and options are given, web_terminal will run bash by default.
You can run shells but also single programs, such as htop. 

***known bug***
On MacOS X, running zsh with web_terminal will produce an extra % 
each time. Consider it a ___feautre___, will not fix unless there is a 
pull request. 


**NOTE**

___Do NOT run this in an untrusted network. You will expose your 
shell to anyone that can access your network and Do NOT leave
the server running.___

Here is a screenshot:

<img src="https://github.com/syssecfsu/web_terminal/blob/master/extra/screenshot.png?raw=true" width="800px">
