class TerminalManager {
    constructor() {
        this.terminals = [];
        this.activeTerminalId = null;
        this.tabsContainer = document.getElementById('tabs-list');
        this.terminalsContainer = document.getElementById('terminals-container');
        this.newTabButton = document.getElementById('new-tab');
        this.terminalList = document.getElementById('terminal-list');

        this.newTabButton.addEventListener('click', () => this.createTerminal());
        this.createTerminal(); // 初始创建第一个终端

        window.addEventListener('resize', () => {
            this.resizeActiveTerminal();
        });
    }

    createTerminal() {
        const id = `term-${Date.now()}`;
        const title = `PowerShell ${this.terminals.length + 1}`;

        // 创建标签页
        const tab = document.createElement('div');
        tab.className = 'tab-item';
        tab.textContent = title;
        tab.dataset.id = id;

        tab.addEventListener('click', () => this.switchTerminal(id));

        // 创建关闭按钮
        const closeBtn = document.createElement('span');
        closeBtn.textContent = ' x';
        closeBtn.style.marginLeft = '10px';
        closeBtn.style.cursor = 'pointer';
        closeBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            this.closeTerminal(id);
        });
        tab.appendChild(closeBtn);

        this.tabsContainer.appendChild(tab);

        // 创建终端容器
        const termContainer = document.createElement('div');
        termContainer.id = id;
        termContainer.className = 'terminal-container';
        this.terminalsContainer.appendChild(termContainer);

        // 初始化xterm
        const term = new Terminal({
            cursorBlink: true,
            theme: {
                background: '#1e1e1e',
                foreground: '#d4d4d4'
            },
            windowsMode: true
        });

        const fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);
        term.open(termContainer);
        fitAddon.fit();

        // 创建WebSocket连接
        const ws = new WebSocket(`ws://${window.location.host}/ws`);

        term.onData(data => {
            // 特殊键处理
            if (data === '\r' || data === '\n') {
                ws.send('\n');
            } else if (data === '\x7f') { // Backspace
                ws.send('\b');
            } else {
                ws.send(data);
                term.write(data);
            }
        });

        ws.onmessage = event => {
            // 将Windows换行符统一转换为Unix换行符

            // 写入终端前重新加入回车符
            term.write(event.data);
        };

        ws.onclose = () => {
            const exitMsg = '\r\n\x1b[31mConnection closed - Exit code: ' +
                (terminalExitCode !== null ? terminalExitCode : 'N/A') +
                '\x1b[0m\r\n';
            term.write(exitMsg);
            // 禁用输入
            term.off('data');
        };

        // // 初始化后发送回车触发提示符
        // setTimeout(() => {
        //     ws.send('\r');
        // }, 500);

        const listItem = document.createElement('li');
        listItem.textContent = title;
        listItem.dataset.id = id;
        listItem.addEventListener('click', () => this.switchTerminal(id));
        this.terminalList.appendChild(listItem);

        this.terminals.push({ id, term, ws, tab, container: termContainer, listItem, fitAddon });
        this.switchTerminal(id);

        return id;
    }

    switchTerminal(id) {
        this.terminals.forEach(t => {
            const isActive = t.id === id;
            t.container.classList.toggle('active', isActive);
            t.tab.classList.toggle('active', isActive);
            t.listItem.classList.toggle('active', isActive);
        });
        this.activeTerminalId = id;

        this.resizeActiveTerminal()
    }

    closeTerminal(id) {
        const index = this.terminals.findIndex(t => t.id === id);
        if (index === -1) return;

        const [terminal] = this.terminals.splice(index, 1);
        terminal.ws.close();
        terminal.term.dispose();
        terminal.tab.remove();
        terminal.container.remove();
        terminal.listItem.remove();

        if (terminal.id === this.activeTerminalId && this.terminals.length > 0) {
            this.switchTerminal(this.terminals[0].id);
        }
    }

    resizeActiveTerminal() {
        if (!this.activeTerminalId) return;

        const terminal = this.terminals.find(t => t.id === this.activeTerminalId);
        if (terminal && terminal.fitAddon) {
            // 延迟执行以确保DOM更新完成
            setTimeout(() => {
                try {
                    terminal.fitAddon.fit();
                } catch (e) {
                    console.log("Resize error:", e);
                }
            }, 50);
        }
    }
}

// 页面加载完成后初始化
window.addEventListener('load', () => {
    window.terminalManager = new TerminalManager();

    // 确保终端在窗口大小变化时自适应
    const resizeObserver = new ResizeObserver(() => {
        document.querySelectorAll('.terminal-container.active .xterm').forEach(termEl => {
            const term = termEl.xterm;
            if (term && term.fitAddon) {
                term.fitAddon.fit();
            }
        });
    });

    resizeObserver.observe(document.getElementById('terminals-container'));
});