Labean is a simple HTTP/HTTPS port knocker for GNU/Linux.

### What is port knocking?
Port knocking is a method of externally opening ports on a firewall by doing some actions, e.g. generating a connection attempt on a set of prespecified closed ports.  [[Wikipedia](https://en.wikipedia.org/wiki/Port_knocking)]

The primary purpose of port knocking is to prevent an attacker from scanning a system for potentially exploitable services by doing a port scan, because unless the attacker sends the correct knock sequence, the protected ports will appear closed.
Another purpose is to disguise some services (e.g. VPN or proxy) running on the server. For example, the server receives connections on 443 port and acts like a usual HTTPS web server, but after knocking it will modify firewall rules to allow specified IP to connect to another service on the same port - for third-party observers (like corporate or ISP Deep-Packet-Inspection systems) it will look exactly like ordinary HTTPS.

### Why Labean?
Classic implementations of port knockers (like [knockd](http://www.zeroflux.org/projects/knock)) allow client open port or start services by generating a connection attempt on a set of prespecified closed ports. This is simple and usually reliable, but there are some tricky cases. First of all, this requires using special clients or scripts on the client device (including mobile gadgets or network routers). More significant problem is that all ports and protocols excepts standard 80 (HTTP) and 443 (HTTPS) can be banned on corporate or ISP firewall, so you can't use 'classic' knocking  in this case. So that's why Labean was created.

### How it works?
Briefly: there is a front-end web server (like [nginx](http://nginx.org/) or [lighttpd](https://www.lighttpd.net/)) running on your VDS/VPS/etc. and it servers some ordinary web content like cute kittens' videos, Linux distros ISOs or Wikipedia mirror. But when you want to connect to the hidden service (VPN, proxy, ssh daemon, etc.), you perfrom GET request (using CURL or common web browser) like:

`https://someserver.org/secret/vpn/on`,

process basic authentication, then your front-end web server reverse-proxies the request to Labean, and Labean starts service or applies firewall to open access exclusively for your IP address. When you finished working with hidden service, you just make GET

`https://someserver.org/secret/vpn/off`

and the access becomes closed.

Another way is to use *automatic timeout control*.

For example, if you only need to open port on firewall temporarly to accept connection, you can set timeout for 30 or 60 seconds. After making GET request you start your client and initiate a connection to the hidden service, then the timeout expires and the rule is deleted. Your established connection is still active, but no one else even from your IP can establish a new one without knocking again.

### Building and installation
Labean is written in Go language, so you'll need to have Golang compiler on your system. Please refer to [official Go website](https://golang.org/doc/install) and your distro manpages to get and install Golang compiler. I tried to build Labean with Go 1.6 and 1.10 and everything was fine, perhaps it will works with even older versions.

``` # clone the latest source code:
git clone https://github.com/uprt/labean.git
cd labean
go build 
```

If everything is okay, you'll have `labean` binary executable in your current directory.

I recommend to put `labean` binary to `/usr/sbin` directory and `labean.conf` to `/etc/` (see the next chapter about this configuration file).

Labean can be started simply:

`./labean <path_to_config_file>`

After starting `labean` will send its logs to syslog service, so you can check them out in /var/log/syslog or /var/log/messages file depending on your distro.

But the better way is to run it as a service. If your distro uses [LSB](https://wiki.debian.org/LSBInitScripts)-compatible init system or [Systemd](https://www.freedesktop.org/wiki/Software/systemd/), feel free to use sample service files that you can find in `examples` dir in the source tree:
```
cp ./examples/labean.init.ex /etc/init.d/labean
/etc/init.d/labean start
update-rc.d labean defaults
# or...
cp ./examples/labean.service.ex /etc/systemd/system/labean.service # this path can be different in your distro
systemctl daemon-reload
systemctl start labean
systemctl enable labean
```

Another important step of Labean installation is a configuration of  your frontend web server (reverse-proxy). Labean does not support SSL, HTTP auth and serving static or dynamic websites itself. I like "UNIX-way" philosophy: "Make each program do one thing well." So we have to use a web server with Labean. It have to perform the following features:

- serve HTTPS (SSL) connections with valid certificates
- require basic HTTP authorization for 'secret' URL
- add 'X-Real-IP" or similar header to HTTP request
- pass request to Labean (runnign on another TCP port like 8080) for 'secret' URL

I recommend to use Nginx for thins things. The simpliest variant of the config is here:
```
server {
  listen 443 ssl http2;
  ssl on;
  server_name testserver.org;
  ssl_certificate /etc/letsencrypt/live/testserver.org/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/testserver.org/privkey.pem;
  root /srv/www/funny_kittens/;
  index index.html;

  location ~ ^/secret/(.*) {
    auth_basic      "Administrator Login";
    auth_basic_user_file  /var/www/.htpasswd;
    proxy_set_header  X-Real-IP  $remote_addr;
    proxy_pass http://127.0.0.1:8080/$1;
  }

  # or, if you would like to use 'url_prefix' setting in Labean config file
  # set "url_prefix" value to "secret"
  # location /secret/ {
  #   auth_basic      "Administrator Login";
  #   auth_basic_user_file  /var/www/.htpasswd;
  #   proxy_set_header  X-Real-IP  $remote_addr;
  #   proxy_pass http://127.0.0.1:8080/;
  # }
}                            
```
You can find it in `examples` directory of the source tree.
Anyway, you can use any other webservers/reverse-proxies if you want.

### Config file
Labean config file is a simple JSON document. 
```
{
  "listen": "127.0.0.1:8080",
  "url_prefix": "secret",
  "external_ip": "192.30.253.113",
  "real_ip_header": "X-Real-IP",
  "tasks": [ 
    {
      "name": "vpn",
      "timeout": 30,
      "on_command": "iptables -t nat -A PREROUTING -p tcp -s {clientIP} --dport 443 -j REDIRECT --to-port 4443",
      "off_command": "iptables -t nat -D PREROUTING -p tcp -s {clientIP} --dport 443 -j REDIRECT --to-port 4443"
    },
    {
      "name": "sshd",
      "timeout": 0,
      "on_command": "/etc/init.d/sshd start",
      "off_command": "/etc/init.d/sshd stop"
    }
    ]
}
```

Here is the description of its fields:
- `"listen"`: IP address and port to listen for incoming connections from reverse proxy;
- `"url_prefix"`: if you reverse-proxy can't rewrite URLs, you can set the prefix to trim;
- `"external_ip"`: IP of your server. This is not 'must-have' option, its just a sugar to replace {serverIP} in the commands strings if you need;
- `"real_ip_header"`: the name of HTTP header with the real client IP added by the reverse-proxy (usually it is "X-Real-IP");
- `"tasks"`: the array of the 'tasks' to start or stop hidden services;
- `"name"`: the unique name of the hidden service. You will use it in your HTTP GET queries: `https://someserver.org/secret/<name>/{on|off}`;
- `"timeout"`: timeout to automatically switch your hidden service or firewall rule "off" after "on". If it is set to 0, it means that timeout feature will be disabled and you need to "off" your service manually;
- `"on_command"`: command line to start your service or activate firewall rule. You can use `{serverIP}` and `{clientIP}` macro in it, they will be automatically replaced by the corresponding values;
- `"off_comand"`: the same as `"on_command"`, but does exactly the opposite thing :)

### Bugs and ideas?
Feel free to open Issues on Github or make pull requests if you want to improve something. I will be grateful for any help and feedback.

### License
Labean has BSD 2-Clause "Simplified" License.

Copyright (c) 2018, Kirill Ovchinnikov

