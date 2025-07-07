class TerminalManager {
    constructor() {
        this.terminals = [];
        this.activeTerminalId = null;
        this.tabsContainer = document.getElementById('tabs-list');
        this.terminalsContainer = document.getElementById('terminals-container');
        this.newTabButton = document.getElementById('new-tab');

        this.newTabButton.addEventListener('click', () => this.createTerminal());
        this.createTerminal(); // 初始创建第一个终端
    }

    createTerminal() {
        const id = `term-${Date.now()}`;

        // 创建标签页
        const tab = document.createElement('div');
        tab.className = 'tab-item';
        tab.textContent = `PowerShell ${this.terminals.length + 1}`;
        tab.dataset.id = id;

        tab.addEventListener('click', () => this.switchTerminal(id));

        // 创建关闭按钮
        const closeBtn = document.createElement('span');
        closeBtn.textContent = ' ×';
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
        term.loadAddon(new WebLinksAddon.WebLinksAddon());
        term.open(termContainer);
        fitAddon.fit();

        // 创建WebSocket连接
        const ws = new WebSocket(`ws://${window.location.host}/ws`);

        term.onData(data => {
            // 特殊键处理
            if (data === '\r') {
                data = '\r\n';
            } else if (data === '\x7f') { // Backspace
                data = '\b \b';
            }
            ws.send(data);
            console.log(data)
            term.write(data)
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

        this.terminals.push({ id, term, ws, tab, container: termContainer });
        this.switchTerminal(id);
    }

    switchTerminal(id) {
        this.terminals.forEach(t => {
            const isActive = t.id === id;
            t.container.classList.toggle('active', isActive);
            t.tab.classList.toggle('active', isActive);
        });
        this.activeTerminalId = id;
    }

    closeTerminal(id) {
        const index = this.terminals.findIndex(t => t.id === id);
        if (index === -1) return;

        const [terminal] = this.terminals.splice(index, 1);
        terminal.ws.close();
        terminal.term.dispose();
        terminal.tab.remove();
        terminal.container.remove();

        if (terminal.id === this.activeTerminalId && this.terminals.length > 0) {
            this.switchTerminal(this.terminals[0].id);
        }
    }
}

// 页面加载完成后初始化
window.addEventListener('load', () => {
    new TerminalManager();

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