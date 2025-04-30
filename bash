mkdir -p ~/.gocli
cat > ~/.gocli/translate.yaml << EOF
api_key: your-api-key
default_source: auto
default_target: zh
cache_enabled: true
cache_duration: 24h
max_concurrency: 5
timeout: 30s
EOF