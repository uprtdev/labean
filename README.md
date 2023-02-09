Labean is a simple HTTP/HTTPS port knocker for GNU/Linux.

### What is port knocking?
Port knocking is a method of externally opening ports on a firewall by doing some actions, e.g. generating a connection attempt on a set of prespecified closed ports.  [[Wikipedia](https://en.wikipedia.org/wiki/Port_knocking)]

The primary purpose of port knocking is to prevent an attacker from scanning a system for potentially exploitable services by doing a port scan, because unless the attacker sends the correct knock sequence, the protected ports will appear closed.
Another purpose is to disguise some services (e.g. VPN or proxy) running on the server. For example, the server receives connections on 443 port and acts as a usual HTTPS web server, but after knocking it will modify firewall rules to allow specified IP to connect to another service on the same port - for third-party observers (like corporate or ISP Deep-Packet-Inspection systems) it will look exactly like ordinary HTTPS.

### Why Labean?
Classic implementations of port knockers (like [knockd](http://www.zeroflux.org/projects/knock)) allow a client to open port or start services by generating a connection attempt on a set of prespecified closed ports. This is simple and usually reliable, but there are some tricky cases. First of all, this requires using special clients or scripts on the client device (including mobile gadgets or network routers). More significant problem is that all ports and protocols except standard 80 (HTTP) and 443 (HTTPS) can be banned on corporate or ISP firewall, so you can't use 'classic' knocking in this case. So that's why Labean was created.

### How it works?
Briefly: there is a front-end web server (like [nginx](http://nginx.org/) or [caddy](https://caddyserver.com/)) running on your VDS/VPS/etc. and it serves some ordinary web content like cute kittens' videos, Linux distros' ISOs or Wikipedia mirror. But when you want to connect to the hidden service (VPN, proxy, ssh daemon, etc.), you perform GET request (using CURL or a web browser) like:

`https://someserver.org/secret/vpn/on`,

process basic authentication, then your front-end web server reverse-proxies the request to Labean, and Labean starts service or applies firewall rules to open access exclusively for your IP address. When you finished working with hidden service, you just make GET

`https://someserver.org/secret/vpn/off`

and the access becomes closed.

Another way is to use *automatic timeout control*.

For example, if you only need to open firewall port temporarily to accept a connection, you can set a timeout for 30 or 60 seconds. After making GET request you start your client and initiate a connection to the hidden service, then the timeout expires and the rule is deleted. Your established connection is still active, but no one else even from your IP can establish a new one without knocking again.

### Building and installation
Labean is written in Go language, so you'll need Golang compiler installed in your system. Please refer to [official Go website](https://golang.org/doc/install) and your distro manpages to get and install Golang compiler. I tried to build Labean with Go 1.6 and 1.10 and everything was fine, perhaps it will work with even older versions.

``` # clone the latest source code:
git clone https://github.com/uprt/labean.git
cd labean
go build 
```

If everything is okay, you'll have `labean` binary executable in your current directory.

I recommend putting `labean` binary to `/usr/sbin` directory and `labean.conf` to `/etc/` (see the next chapter about this configuration file).

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

Another important step of Labean installation is a configuration of your frontend web server (reverse-proxy). Labean does not support SSL, HTTP auth and serving static or dynamic websites itself. I like "UNIX-way" philosophy: "Make each program do one thing well." So we have to use a web server with Labean. It should perform the following things:

- serve HTTPS (SSL) connections with valid certificates
- require basic HTTP authorization for 'secret' URL
- add 'X-Real-IP" or similar header to an HTTP request
- pass the request to Labean (running on another TCP port like 8080) for 'secret' URL

I recommend using Caddy or Nginx for this.
The simplest variant of the config you can find it in `examples` directory of the source tree.
Anyway, you can use any other webservers/reverse-proxies if you want.

### Config file
Labean config file is a simple JSON document. 
```
{
  "listen": "127.0.0.1:8080",
  "url_prefix": "secret",
  "external_ip": "192.30.253.113",
  "external_ipv6": "9c49:226f:fd31:bc73:4c84:29ed:13aa:db07",
  "real_ip_header": "X-Real-IP",
  "allow_explicit_ips": false,
  "tasks": [ 
    {
      "name": "vpn",
      "timeout": 30,
      "on_command": "iptables -t nat -A PREROUTING -p tcp -s {clientIP} --dport 443 -j REDIRECT --to-port 4443",
      "off_command": "iptables -t nat -D PREROUTING -p tcp -s {clientIP} --dport 443 -j REDIRECT --to-port 4443",
      "on_command_v6": "ip6tables -t nat -A PREROUTING -p tcp -s {clientIP} --dport 443 -j REDIRECT --to-port 4443",
      "off_command_v6": "ip6tables -t nat -D PREROUTING -p tcp -s {clientIP} --dport 443 -j REDIRECT --to-port 4443"
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
- `"listen"`: IP address and port to listen for incoming connections from reverse proxy; 127.0.0.1 for IPv4 localhost, ::1 for IPv6 localhost, and so on;
- `"url_prefix"`: if your reverse-proxy can't rewrite URLs, you can set the prefix to trim; if your reverse-proxy performs trimming, leave it empty;
- `"external_ip"`: IP of your server. This is not 'must-have' option, its just a sugar to replace {serverIP} in the commands strings if you need;
- `"external_ipv6"`: The same, but for IPv6 (if you use it)
- `"real_ip_header"`: the name of HTTP header with the real client IP added by the reverse-proxy (usually it is "X-Real-IP" or "X-Forwarded-For"); leave it empty if you don't use reverse-proxy;
- `"allow_explicit_ips"`: if true, you can explicitly specify client IP address in GET request, like `https://someserver.org/secret/service/off/?ip=123.56.78.9`. This can be helpful when you established VPN connection and later want to manually de-activate secret service using Labean's internal (local) IP inside the tunnel. This feature allows you to stop services started by other users, so use it very carefully, and it is disabled by default.
- `"tasks"`: the array of the 'tasks' to start or stop hidden services;
- `"name"`: the unique name of the hidden service. You will use it in your HTTP GET queries: `https://someserver.org/secret/<name>/{on|off}`;
- `"timeout"`: timeout to automatically switch your hidden service or firewall rule "off" after "on". If it is set to 0, it means that timeout feature will be disabled and you need to "off" your service manually;
- `"on_command"`: command line to start your service or activate firewall rule. You can use `{serverIP}` and `{clientIP}` macro in it, they will be automatically replaced by the corresponding values;
- `"off_command"`: the same as `"on_command"`, but does exactly the opposite thing :)
- `"on_command_v6"` and `"off_command_v6"`: same as above, but for IPv6 (if you use it)

### Bugs and ideas?
Feel free to open Issues on Github or make pull requests if you want to improve something. I will be grateful for any help and feedback.

### License
Labean has BSD 2-Clause "Simplified" License.

Copyright (c) 2018-2023, Kirill Ovchinnikov

