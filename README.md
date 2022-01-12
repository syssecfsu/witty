# WiTTY: Web-based interactive TTY
This program allows you to use terminal in the browser. Simply run the program and give it the command to execute when users connect via the browser. ___Interestingly___, it allows others to view your interactive sessions as well. This could be useful to provide remote support and/or help. You can use the program to run any command line programs, such as ```bash```, ```htop```, ```vi```, ```ssh```. This following screenshot shows that six interactive session running ```zsh``` on macOS Monterey. <img src="https://github.com/syssecfsu/witty/blob/master/extra/main.png?raw=true" width="800px">

To use the program, you need to provide a TLS cert. You can request a free [Let's Encrypt](https://letsencrypt.org/) cert or use a self-signed cert. The program currently does not support user authentication. Therefore, do not run it in untrusted networks or leave it running. A probably safe use of the program is to run ```ssh```. Please ensure that you do not automatically login to the ssh server (e.g., via key authentication).

___AGAIN, Do NOT run this in an untrusted network. You will expose your 
shell to anyone that can access your network and Do NOT leave
the server running.___

This program is written in the [go programming language](https://go.dev/), using the 
[Gin web framework](https://github.com/gin-gonic/gin), [gorilla/websocket](https://github.com/gorilla/websocket), [pty](https://github.com/creack/pty), and the wonderful [xterm.js](https://xtermjs.org/)!
The workflow is simple, the client will initiate a terminal 
window (xterm.js) and create a websocket with the server, which relays the data between pty and xterm. You can customize the look and feel of the HTML pages by editing files under the ```assets``` directory.


## Installation

1. Install the [go](https://go.dev/) compiler.
2. Download the release and unzip it, or clone the repo
   
   ```git clone https://github.com/syssecfsu/witty.git```

3. Go to the ```tls``` directory and create a self-signed cert
   
   \# Generate a private key for a curve

    ```openssl ecparam -name prime256v1 -genkey -noout -out private-key.pem```

    \# Create a self-signed certificate

    ```openssl req -new -x509 -key private-key.pem -out cert.pem -days 360```

4. Return to the root directory of the source code and build the program
   
   ```go build .```

5. Start the server and give it the command to run. The server listens on 8080, for example:
   
   ```./witty htop``` or

   ```./witty ssh <your_server_ip> -l <user_name>```

6. Connect to the server, for example

   ```https://your_ip_address:8080```

The program has been tested on Linux, WSL2, Raspberry Pi 3B (Debian), and MacOSX using Google Chrome, Firefox, and Safari.

## An Screencast <sub>featuring an older version of WiTTY</sub>

Here is a screencast for sshing into Raspberry Pi running 
[pi-hole](https://pi-hole.net/) 
(```./witty ssh 192.168.1.2 -l pi```,
WiTTY runs in a WSL2 VM on Windows):

<img src="https://github.com/syssecfsu/witty/blob/master/extra/screencast.gif?raw=true" width="800px">
