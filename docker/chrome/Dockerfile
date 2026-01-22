# Chrome 远程调试容器
FROM alpine:latest

# 安装 Chromium 和必要的依赖
RUN apk add --no-cache \
    chromium \
    chromium-chromedriver \
    harfbuzz \
    nss \
    freetype \
    ttf-freefont \
    font-noto-emoji \
    wqy-zenhei \
    && rm -rf /var/cache/apk/*

# 创建用户（避免以 root 运行）
RUN addgroup -g 1000 chrome && \
    adduser -D -u 1000 -G chrome chrome && \
    mkdir -p /home/chrome/Downloads /home/chrome/.config/chromium && \
    chown -R chrome:chrome /home/chrome

# 切换到 chrome 用户
USER chrome
WORKDIR /home/chrome

# 暴露远程调试端口
EXPOSE 9222

# 启动 Chromium
# --no-sandbox: 禁用沙箱（容器内必需）
# --disable-dev-shm-usage: 避免共享内存问题
# --remote-debugging-address=0.0.0.0: 允许外部访问
# --remote-debugging-port=9222: 调试端口
# --disable-gpu: 禁用 GPU（服务器环境通常需要）
# --no-first-run: 跳过首次运行向导
# --no-default-browser-check: 跳过默认浏览器检查
# --disable-extensions: 禁用扩展
# --disable-background-networking: 禁用后台网络
CMD ["chromium-browser", \
     "--no-sandbox", \
     "--disable-dev-shm-usage", \
     "--disable-gpu", \
     "--remote-debugging-address=0.0.0.0", \
     "--remote-debugging-port=9222", \
     "--no-first-run", \
     "--no-default-browser-check", \
     "--disable-extensions", \
     "--disable-background-networking", \
     "--disable-background-timer-throttling", \
     "--disable-backgrounding-occluded-windows", \
     "--disable-breakpad", \
     "--disable-component-extensions-with-background-pages", \
     "--disable-features=TranslateUI", \
     "--disable-ipc-flooding-protection", \
     "--disable-renderer-backgrounding", \
     "--enable-features=NetworkService,NetworkServiceInProcess", \
     "--force-color-profile=srgb", \
     "--hide-scrollbars", \
     "--metrics-recording-only", \
     "--mute-audio", \
     "--user-data-dir=/home/chrome/.config/chromium", \
     "about:blank"]

