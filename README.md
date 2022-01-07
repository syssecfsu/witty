# Web Terminal
A (unsafe) technical demo to export a shell to web browser. 
It is just a simple demo in case some people are interested in 
how to setup xterm.js with websocket. 

This program is written in the go programming language, using the 
Gin web framework, gorilla/websocket, pty, and xterm.js!
The workflow is simple, the client will initiate a terminal 
window (xterm.js) and create a websocket with the server. On 
the server side, it serves the basic HTML/JS/CSS files and 
websockets (by shovling the data between pty and xterm).

___It is amazing what you can do with 270 lines of go code.___ 

To use the program, download/clone the code, and in the web_terminal
directory, run ```go build .```, this will create the binary called
web_terminal. Then, go to the tls directory and create a self-signed
certificate according to the instructions in README.

To run it, use ```./web_terminal cmd options_to_cmd```.
If no cmd and options are given, web_terminal will run bash by default.
You can run shells but also single programs, such as htop. For example, 
you can export the ssh shell, such as ```./web_terminal ssh 192.168.1.2 -l pi```.



The program
has been tested on Linux, WSL2, Raspberry Pi 3B (Debian), and MacOSX.

***known bug***

On MacOS X, running zsh with web_terminal will produce an extra % 
each time. Consider it a ___feautre___, will not fix unless there is a 
pull request. 


**NOTE**

___Do NOT run this in an untrusted network. You will expose your 
shell to anyone that can access your network and Do NOT leave
the server running.___

Here is a screencast for sshing into Raspberry Pi running 
[pi-hole](https://pi-hole.net/) (```./web_terminal ssh 192.168.1.2 -l pi```):

<img src="https://github.com/syssecfsu/web_terminal/blob/master/extra/screencast.gif?raw=true" width="800px">
