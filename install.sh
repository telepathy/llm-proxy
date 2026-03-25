#!/bin/bash
set -e

REPO="telepathy/llm-proxy"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/llm-proxy"
SERVICE_NAME="llm-proxy"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        error "请使用 root 权限运行此脚本: sudo $0"
    fi
}

detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64)  echo "amd64" ;;
        aarch64) echo "arm64" ;;
        arm64)   echo "arm64" ;;
        *)       error "不支持的架构: $arch" ;;
    esac
}

detect_os() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    echo "$os"
}

get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version=$(curl -s "$api_url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        error "无法获取最新版本信息"
    fi
    echo "$version"
}

download_binary() {
    local version=$1
    local os=$2
    local arch=$3
    local binary_name="llm-proxy-${os}-${arch}"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"
    
    if [ "$os" = "windows" ]; then
        binary_name="${binary_name}.exe"
    fi
    
    info "下载 ${binary_name}..."
    info "URL: ${download_url}"
    
    curl -L -o "/tmp/${binary_name}" "$download_url"
    
    if [ $? -ne 0 ]; then
        error "下载失败"
    fi
    
    mv "/tmp/${binary_name}" "${INSTALL_DIR}/llm-proxy"
    chmod +x "${INSTALL_DIR}/llm-proxy"
}

collect_config() {
    echo ""
    echo "=== 配置信息 ==="
    
    read -p "OpenRouter API Key: " openrouter_key
    if [ -z "$openrouter_key" ]; then
        error "API Key 不能为空"
    fi
    
    read -p "代理用户名 [admin]: " username
    username=${username:-admin}
    
    read -sp "代理密码: " password
    echo ""
    if [ -z "$password" ]; then
        error "密码不能为空"
    fi
    
    read -p "监听端口 [8080]: " port
    port=${port:-8080}
    
    read -p "是否启用 HTTPS? (y/N): " enable_https
    enable_https=${enable_https:-n}
}

generate_config() {
    mkdir -p "$CONFIG_DIR"
    
    cat > "${CONFIG_DIR}/config.yaml" << EOF
server:
  host: "0.0.0.0"
  port: ${port}
  read_timeout: 120s
  write_timeout: 120s

auth:
  enabled: true
  users:
    - username: "${username}"
      password: "${password}"

api_keys:
  openrouter: "${openrouter_key}"

default_backend: "openrouter"

routes:
  - model_pattern: "*"
    backend: "openrouter"

backends:
  openrouter:
    type: "openrouter"
    base_url: "https://openrouter.ai/api/v1"
    api_key: "\${api_keys.openrouter}"
    extra_headers:
      HTTP-Referer: "https://github.com/telepathy/llm-proxy"
      X-Title: "LLM Proxy"

logging:
  level: "info"
  format: "json"
EOF
    
    chmod 600 "${CONFIG_DIR}/config.yaml"
    info "配置文件已生成: ${CONFIG_DIR}/config.yaml"
}

create_service() {
    cat > "/etc/systemd/system/${SERVICE_NAME}.service" << EOF
[Unit]
Description=LLM API Proxy Service
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
ExecStart=${INSTALL_DIR}/llm-proxy -config ${CONFIG_DIR}/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"
    
    info "Systemd 服务已创建并启动"
}

verify_installation() {
    sleep 2
    
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        info "服务运行正常"
        info "状态: systemctl status ${SERVICE_NAME}"
        info "日志: journalctl -u ${SERVICE_NAME} -f"
    else
        warn "服务未正常启动，请检查日志:"
        echo "  journalctl -u ${SERVICE_NAME} -n 50"
    fi
}

main() {
    echo "=== LLM Proxy 安装脚本 ==="
    echo ""
    
    check_root
    
    local os=$(detect_os)
    local arch=$(detect_arch)
    info "检测到系统: ${os}/${arch}"
    
    local version=$(get_latest_version)
    info "最新版本: ${version}"
    
    collect_config
    
    download_binary "$version" "$os" "$arch"
    generate_config
    create_service
    verify_installation
    
    echo ""
    echo "=== 安装完成 ==="
    echo "API 地址: http://$(hostname -I | awk '{print $1}'):${port}/v1"
    echo "认证方式: Basic Auth (${username}:${password})"
    echo ""
    echo "测试命令:"
    echo "  curl -u ${username}:${password} http://localhost:${port}/v1/models"
}

main "$@"
