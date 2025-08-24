# Setup Guide: Go Links with pfSense and Raspberry Pi

This guide details how to set up the **Go Links** server in a home network environment using a _Raspberry Pi_ for hosting and a _pfSense_ router for DNS management.

The end goal is to access your links from any device on your local network by simply typing `http://go/<alias>` into your browser.

### Architecture Overview

1.  **Client Device**: Makes a request to `http://go/some-alias`.
2.  **pfSense DNS Resolver**: Intercepts the request for the host `go` and resolves it to your _Raspberry Pi_'s local IP address.
3.  **Raspberry Pi (Nginx)**:
    - Receives the HTTP request on port 80.
    - Redirects the request from HTTP to HTTPS (port 443).
    - Acts as a reverse proxy, forwarding the secure request to the Go Links application running on `localhost:3000`.
4.  **Go Links Application**: Looks up the alias and issues a redirect to the final destination URL.

---

### Step 1: Configure pfSense DNS Override

First, we need to tell your network that the hostname `go` should point to your _Raspberry Pi_.

1.  **Find your Raspberry Pi's IP Address**: Make sure your _Raspberry Pi_ has a static IP address or a DHCP reservation in _pfSense_ so its IP doesn't change. You can find its current IP by running `hostname -I` on the _Pi_. Let's assume the IP is `192.168.1.10`.

2.  **Log into pfSense**: Open your _pfSense_ web interface.

3.  **Navigate to DNS Resolver**: Go to **Services > DNS Resolver**.

4.  **Add a Host Override**: Scroll down to the `Host Overrides` section and click **Add**.

5.  **Fill in the details**:

    - **Host**: `go`
    - **Domain**: `your.local.domain` (e.g., `lan` or `home.arpa`. This should match your pfSense domain in **System > General Setup**).
    - **IP Address**: `192.168.1.10` (Your _Raspberry Pi_'s IP).
    - **Description**: `Go Links Raspberry Pi`.

6.  **Save** the Host Override and then **Apply Changes**.

Now, any device on your network that uses _pfSense_ router for DNS will resolve `go.your.local.domain` (and just `go` in most cases) to `192.168.1.10`.

---

### Step 2: Configure Raspberry Pi (Nginx Reverse Proxy)

We'll set up _Nginx_ to handle incoming requests, enforce HTTPS, and forward traffic to our Go application.

1.  **Install Nginx**:

    ```bash
    sudo apt update
    sudo apt install nginx -y
    ```

2.  **Generate a Self-Signed SSL Certificate**: For a local service, a self-signed certificate is sufficient to enable HTTPS.

    ```bash
    # Create a directory for the certs
    sudo mkdir -p /etc/nginx/ssl

    # Generate the key and certificate
    sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout /etc/nginx/ssl/nginx-selfsigned.key \
        -out /etc/nginx/ssl/nginx-selfsigned.crt
    ```

    You will be prompted to fill in some information. Since it's for local use, the details don't matter much.

3.  **Create an Nginx Configuration File**:
    Create a new config file for your **Go Links** service.

    ```bash
    sudo nano /etc/nginx/sites-available/go-links
    ```

    Paste the following configuration into the file. This config does two things:

    - Listens on port 80 (HTTP) and permanently redirects to HTTPS.
    - Listens on port 443 (HTTPS), uses the SSL certificate, and proxies requests to the Go app.

    ```nginx
    # Redirect HTTP to HTTPS
    server {
        listen 80;
        server_name go; # Or your Pi's IP
        return 301 https://$host$request_uri;
    }

    # Main server block for proxying
    server {
        listen 443 ssl;
        server_name go; # Or your Pi's IP

        # SSL Certificate
        ssl_certificate /etc/nginx/ssl/nginx-selfsigned.crt;
        ssl_certificate_key /etc/nginx/ssl/nginx-selfsigned.key;

        location / {
            proxy_pass http://localhost:3000;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
    ```

4.  **Enable the Site and Test**:

    ```bash
    # Create a symbolic link to enable the site
    sudo ln -s /etc/nginx/sites-available/go-links /etc/nginx/sites-enabled/

    # Remove the default config if it conflicts
    sudo rm /etc/nginx/sites-enabled/default

    # Test the Nginx configuration for syntax errors
    sudo nginx -t

    # If the test is successful, reload Nginx
    sudo systemctl reload nginx
    ```

---

### Step 3: Run the Go Links Server

With the proxy in place, the final step is to run the application itself.

1.  **Clone and Run**: On your Raspberry Pi, clone the repository and run the server.

    ```bash
    # Clone your repo
    git clone https://github.com/DmitriiSer/go-links.git
    cd go-links

    # Install dependencies
    go mod tidy

    # Run the server
    go run main.go
    ```

    The server will start listening on `localhost:3000`.

2.  **(Optional) Run as a Service**: For long-term use, you should run the application as a `systemd` service so it starts automatically on boot.

    - Create a service file: `sudo nano /etc/systemd/system/golinks.service`
    - Paste this configuration (adjust `User` and `WorkingDirectory`):

      ```ini
      [Unit]
      Description=Go Links Server
      After=network.target

      [Service]
      User=pi
      WorkingDirectory=/home/pi/go-links
      ExecStart=/usr/local/go/bin/go run main.go
      Restart=always

      [Install]
      WantedBy=multi-user.target
      ```

    - Enable and start the service:
      ```bash
      sudo systemctl enable golinks.service
      sudo systemctl start golinks.service
      ```

---

### You're Done!

You should now be able to go to any browser on your network and type `http://go/g` or `http://go/github`, and it will be securely proxied through Nginx and redirected by the Go Links application. Because you are using a self-signed certificate, your browser will show a privacy warning the first time you visit, which you can safely accept.
