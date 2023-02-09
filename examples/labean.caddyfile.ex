# please set 'real_ip_header' to 'X-Forwarded-For' in Labean config
yourdomain.com {
    # also set "url_prefix" to the address below, e.g. "secretUrl"
    handle /secretUrl/* {
        reverse_proxy http://127.0.0.1:8080
        basicauth {
            # see https://caddyserver.com/docs/caddyfile/directives/basicauth
		    Bob JDJhJDEwJEVCNmdaNEg2Ti5iejRMYkF3MFZhZ3VtV3E1SzBWZEZ5Q3VWc0tzOEJwZE9TaFlZdEVkZDhX
        }
    }	
    handle {
        root * /home/your_static_files_dir_path
        file_server
  }
}
