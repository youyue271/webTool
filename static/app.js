class TerminalManager {
    constructor() {
        // 定义各种部件

        // 执行终端逻辑部件
        this.terminals = [];
        this.activeTerminalId = null;

        // 右侧执行终端tab列表
        this.tabsContainer = document.getElementById('tabs-list');
        this.terminalsContainer = document.getElementById('terminals-container');
        this.newTabButton = document.getElementById('new-tab');

        // sidebar执行终端列表显示
        this.terminalList = document.getElementById('terminal-list');

        // sidebar左下管理终端
        // this.toolManager = new TerminalManager();
        this.initAdminTerminal();

        // 右侧tab滚动显示组件
        this.tabsWrapper = document.querySelector('.tabs-wrapper');
        this.scrollLeftBtn = this.createScrollButton('left');
        this.scrollRightBtn = this.createScrollButton('right');

        // 右侧tab滚动显示组件事件
        // this.tabsWrapper.addEventListener('scroll', () => this.updateScrollIndicator());
        this.scrollLeftBtn.addEventListener('click', () => this.scrollTabs(-150));
        this.scrollRightBtn.addEventListener('click', () => this.scrollTabs(150));
        this.tabsWrapper.addEventListener('wheel', (e) => {
            if (e.deltaY !== 0) {
                // 防止垂直滚动时页面滚动
                e.preventDefault();

                // 将垂直滚动转换为水平滚动
                this.tabsWrapper.scrollLeft += e.deltaY * 2;
                // this.updateScrollIndicator();
            }
        }, {passive: false});

        // 右侧tab栏新建Tab事件
        this.newTabButton.addEventListener('click', () => this.createTerminal());
        this.createTerminal(); // 初始创建第一个终端

        // 窗口大小动态调整
        window.addEventListener('resize', () => {
            this.resizeActiveTerminal();
            this.resizeAdminTerminal();
        });
    }

    // 右上标签页Tab栏

    /**
     * 创建Tab栏滚动按钮
     * @param direction
     * @returns {HTMLDivElement}
     */
    createScrollButton(direction) {
        const button = document.createElement('div');
        button.className = `scroll-button ${direction}`;
        button.innerHTML = direction === 'left' ? '&lt;' : '&gt;';
        document.querySelector('.tabs-container').appendChild(button);
        return button;
    }

    // /**
    //  * 检查Tab标签页是否溢出，配合滚动栏使用
    //  */
    // checkTabOverflow() {
    //     this.updateScrollIndicator();
    // }
    //
    // /**
    //  * 更新滚动指示器状态
    //  */
    // updateScrollIndicator() {
    //     const scrollLeft = this.tabsWrapper.scrollLeft;
    //     const scrollWidth = this.tabsWrapper.scrollWidth;
    //     const clientWidth = this.tabsWrapper.clientWidth;
    //
    //     // 更新滚动指示器位置
    //     const scrollRatio = scrollLeft / (scrollWidth - clientWidth);
    //     const gradientPercent = Math.max(0, Math.min(100, Math.round(scrollRatio * 100)));
    //
    //     // 动态调整指示器渐变方向
    //     const gradientStart = Math.max(0, 100 - gradientPercent - 30);
    //     this.scrollIndicator.style.background =
    //         `linear-gradient(to right, transparent ${gradientStart}%, #252526 ${gradientStart + 30}%)`;
    // }


    /**
     * 滚动标签页
     * @param distance
     */
    scrollTabs(distance) {
        const newPosition = this.tabsWrapper.scrollLeft + distance;
        const maxScroll = this.tabsWrapper.scrollWidth - this.tabsWrapper.clientWidth;

        // 限制在有效范围内
        this.tabsWrapper.scrollLeft = Math.max(0, Math.min(maxScroll, newPosition));
        // this.updateScrollIndicator();
    }

    /**
     * 如果直接切换的话，滚动标签页，确保ActiveTab在滚动栏中显示
     */
    scrollToActiveTab() {
        const activeTab = this.tabsContainer.querySelector('.tab-item.active');
        if (!activeTab) return;

        const wrapperRect = this.tabsWrapper.getBoundingClientRect();
        const tabRect = activeTab.getBoundingClientRect();

        // 检查标签是否在视图外
        if (tabRect.left < wrapperRect.left) {
            // 标签在左边界外
            this.tabsWrapper.scrollLeft += tabRect.left - wrapperRect.left;
        } else if (tabRect.right > wrapperRect.right) {
            // 标签在右边界外
            this.tabsWrapper.scrollLeft += tabRect.right - wrapperRect.right;
        }

        // this.updateScrollIndicator();
    }

    // 执行终端相关

    /**
     * 创建新的执行终端， 并与后端go绑定事件
     * @returns {string} 终端ID
     */
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
        // setTimeout(() => this.checkTabOverflow(), 100);

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
            // windowsMode: true,
            convertEol: true,
        });

        const fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);
        term.open(termContainer);
        fitAddon.fit();

        // 创建WebSocket连接
        const ws = new WebSocket(`ws://${window.location.host}/ws-exec-terminal`);
        term.prompt = () => {
            const prompt = "\rPS> "
            term.write(prompt);
        }
        term.prompt()
        term.onData(data => {
            // 特殊键处理
            if (data === '\r' || data === '\n') {
                ws.send('\n');
                term.write('\n')
                term.prompt()
            } else if (data === '\x7f') { // Backspace
                ws.send('\x7f');
                term.write('\b \b')
            } else {
                ws.send(data);
                term.write(data);
            }
        });

        ws.onmessage = event => {
            // 将Windows换行符统一转换为Unix换行符

            // 写入终端前重新加入回车符
            term.write('\r');
            term.write(event.data);
            term.prompt()
        };

        // ws.onclose = () => {
        //     const exitMsg = '\r\n\x1b[31mConnection closed - Exit code: ' +
        //         (terminalExitCode !== null ? terminalExitCode : 'N/A') +
        //         '\x1b[0m\r\n';
        //     term.write(exitMsg);
        //     // 禁用输入
        //     term.off('data');
        // };

        // // 初始化后发送回车触发提示符
        // setTimeout(() => {
        //     ws.send('\r');
        // }, 500);

        const listItem = document.createElement('li');
        listItem.textContent = title;
        listItem.dataset.id = id;
        listItem.addEventListener('click', () => this.switchTerminal(id));
        this.terminalList.appendChild(listItem);

        this.terminals.push({
            id,
            term,
            ws,
            tab,
            container: termContainer,
            listItem, fitAddon
        });
        this.switchTerminal(id);

        return id;
    }

    /**
     * 执行终端Tab页切换
     * @param id 终端ID
     */
    switchTerminal(id) {
        this.terminals.forEach(t => {
            const isActive = t.id === id;
            t.container.classList.toggle('active', isActive);
            t.tab.classList.toggle('active', isActive);
            t.listItem.classList.toggle('active', isActive);
        });
        this.activeTerminalId = id;

        setTimeout(() => this.scrollToActiveTab(), 50);

        this.resizeActiveTerminal()
    }

    /**
     * 关闭执行终端
     * @param id 终端ID
     */
    closeTerminal(id) {
        const index = this.terminals.findIndex(t => t.id === id);
        if (index === -1) return;

        const [terminal] = this.terminals.splice(index, 1);
        terminal.ws.close();
        terminal.term.dispose();
        terminal.tab.remove();
        terminal.container.remove();
        terminal.listItem.remove();

        // setTimeout(() => this.checkTabOverflow(), 100);

        if (terminal.id === this.activeTerminalId && this.terminals.length > 0) {
            this.switchTerminal(this.terminals[0].id);
        }
    }

    /**
     * resize执行终端
     */
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


    // 管理终端有关
    /**
     * 初始化管理终端
     */
    initAdminTerminal() {
        const container = document.getElementById('admin-terminal');

        const adminTerm = new Terminal({
            convertEol: true,
            cursorBlink: true,
            // rows: 8,
            // cols: 40,
            // windowsMode: true,
            theme: {
                background: '#1e1e1e',
                foreground: '#d4d4d4'
            },
        });


        const fitAddon = new FitAddon.FitAddon();
        adminTerm.loadAddon(fitAddon);
        adminTerm.open(container);
        fitAddon.fit();

        const adminWs = new WebSocket(`ws://${window.location.host}/ws-admin-terminal`);
        adminTerm.onData(data => {
            // 输出发送后端
            if (data === '\r' || data === '\n') {
                adminTerm.write('\n\r')
            } else if (data === '\x7f') { // Backspace
                adminTerm.write('\b \b')
            } else {
                adminTerm.write(data);
            }
            if (adminWs.readyState === WebSocket.OPEN) {
                adminWs.send(data);
            }
        })

        // adminWs.onmessage = event => {
        //     // 显示控制台输出
        //     adminTerm.write(event.data);
        // }

        adminWs.onmessage = event => {
            // 后端创建工具逻辑 TODO：需要定制一下， 这个就是粗略的模式
            const message = event.data;
            if (message.startsWith('CREATE_TOOL:')) {
                const toolInfo = message.split('::',2)[1];
                const [toolPath, toolName, command] = toolInfo.split('|');
                this.createToolTerminal(toolPath, toolName, command);
            } else {
                adminTerm.write(message);
            }
        };

        this.adminTerm = {
            adminTerm,
            adminWs,
            fitAddon
        }
    }

    resizeAdminTerminal() {
        if (this.adminTerm && this.adminTerm.fitAddon) {
            setTimeout(() => {
                try {
                    this.adminTerm.fitAddon.fit();
                } catch (e) {
                    console.log("Resize error:", e);
                }
            }, 50);
        }

    }

    /**
     * 通过控制台创建执行终端tab
     * @param toolPath 工具路径, 后端会先CD过去
     * @param toolName 工具名称, 命名为Tab名
     * @param command 指令(工具地址,通过后端获得)
     */
    createToolTerminal(toolPath, toolName, command) {
        const id = `tool-term-${Date.now()}`;

        // 创建标签页
        const tab = document.createElement('div');
        tab.className = 'tab-item';
        tab.textContent = toolName;
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
        // setTimeout(() => this.checkTabOverflow(), 100);

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
            // windowsMode: true,
            convertEol: true,
        });

        const fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);
        term.open(termContainer);
        fitAddon.fit();

        const listItem = document.createElement('li');
        listItem.textContent = toolName;
        listItem.dataset.id = id;
        listItem.addEventListener('click', () => this.switchTerminal(id));
        this.terminalList.appendChild(listItem);

        const ws = new WebSocket(`ws://${window.location.host}/ws-tool-terminal?exePath=${toolPath}&terminalId=${id}&command=${encodeURIComponent(command)}`);
        term.prompt = () => {
            const prompt = "\rPS> "
            term.write(prompt);
        }
        term.prompt()
        term.write(command);
        term.write('\n');


        term.onData(data => {
            if (data === '\r' || data === '\n') {
                ws.send('\n');
                term.write('\n')
                term.prompt()
            } else if (data === '\x7f') { // Backspace
                ws.send('\x7f');
                term.write('\b \b')
            } else {
                ws.send(data);
                term.write(data);
            }
        });

        ws.onmessage = event => {
            term.write("\r   \r");
            term.write(event.data);
            term.prompt()
        };

        ws.onclose = () => {
            term.write('\r\n\x1b[31m连接关闭\x1b[0m\r\n');
        };

        // 添加到终端列表
        this.terminals.push({
            id,
            term,
            ws,
            tab,
            container: termContainer,
            listItem,
            fitAddon,
            toolName
        });

        // 切换到新创建的终端
        this.switchTerminal(id);
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