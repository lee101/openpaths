upstream openpath_backend {
    server 127.0.0.1:8092;
    keepalive 32;
}

server {
    listen 80;
    server_name openpaths.io;

    proxy_connect_timeout 600s;
    proxy_send_timeout 600s;
    proxy_read_timeout 600s;
    send_timeout 600s;

    client_max_body_size 10M;

    root /nvme0n1-disk/code/openpaths/dist;

    # API and backend routes proxy to Go server
    location ~ ^/(v1|auth|account|health|stats|crypto|stripe|admin|uploads|openrouter)(/|$) {
        proxy_pass http://openpath_backend;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_buffering off;
    }

    # Sitemap index served by Go backend
    location = /sitemap.xml {
        proxy_pass http://openpath_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Art-image search sitemaps (paged) + static-pages sitemap, served by Go backend
    location ~ ^/sitemap-[a-z0-9-]+\.xml$ {
        proxy_pass http://openpath_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Art detail + tag pages proxy to Go so crawler-visible meta (title/og:image)
    # is injected per prompt; the SPA still hydrates client-side.
    location ~ ^/art/(i|tag)/ {
        proxy_pass http://openpath_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SPA route that collides with the public/blog asset directory
    location = /blog {
        try_files /index.html =404;
    }

    location = /blog/ {
        try_files /index.html =404;
    }

    # Static assets from dist
    location /assets/ {
        # Stable (non-hashed) filenames: cache the bytes but ALWAYS revalidate so a
        # new bundle is picked up immediately after a deploy. ETag -> 304 when unchanged.
        add_header Cache-Control "public, max-age=0, must-revalidate" always;
        etag on;
        try_files $uri =404;
    }

    # Everything else serves the SPA
    location / {
        try_files $uri $uri/ /index.html;
    }
}
