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
}
                                  
