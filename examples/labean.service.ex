[Unit]
Description=Labean HTTP port knocker
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/sbin/labean /etc/labean.conf
Restart=on-failure

[Install]
WantedBy=multi-user.target
