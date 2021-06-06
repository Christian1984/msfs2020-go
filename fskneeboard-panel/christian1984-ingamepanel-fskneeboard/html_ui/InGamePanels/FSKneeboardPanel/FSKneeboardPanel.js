let MyTemplateElement;
let myCheckAutoload;

try {
    MyTemplateElement = TemplateElement;
}
catch (e) {
    if (parent && parent.window && parent.window.test_environment) {
        MyTemplateElement = class extends HTMLElement {
            constructor(args) {
                super(args);
            }
    
            connectedCallback() {
    
            }
    
            disconnectedCallback() {
                
            }
        }
    }
}

try {
    myCheckAutoload = checkAutoload;
}
catch (e) {
    if (parent && parent.window && parent.window.test_environment) {
        myCheckAutoload = function() {}
    }
}

class IngamePanelFSKneeboardPanel extends MyTemplateElement {
    constructor() {
        super(...arguments);

        this.panelActive = false;
        this.started = false;
        this.ingameUi = null;
        this.busy = false;
        this.debugEnabled = false;

        this.collapsed = false;
        this.collapse_hotkey = -1;

        if (this.debugEnabled) {
            var self = this;
            setTimeout(() => {
                self.isDebugEnabled();
            }, 1000);
        } else {
            this.initialize();
        }
    }
    isDebugEnabled() {
        var self = this;
        if (typeof g_modDebugMgr != "undefined") {
            g_modDebugMgr.AddConsole(null);
            g_modDebugMgr.AddDebugButton("Identifier", function() {
                console.log('Identifier');
                console.log(self.instrumentIdentifier);
            });
            g_modDebugMgr.AddDebugButton("TemplateID", function() {
                console.log('TemplateID');
                console.log(self.templateID);
            });
            g_modDebugMgr.AddDebugButton("Source", function() {
                console.log('Source');
                console.log(window.document.documentElement.outerHTML);
            });
            g_modDebugMgr.AddDebugButton("close", function() {
                console.log('close');
                if (self.ingameUi) {
                    console.log('ingameUi');
                    self.ingameUi.closePanel();
                }
            });
            this.initialize();
        } else {
            Include.addScript("/JS/debug.js", function () {
                if (typeof g_modDebugMgr != "undefined") {
                    g_modDebugMgr.AddConsole(null);
                    g_modDebugMgr.AddDebugButton("Identifier", function() {
                        console.log('Identifier');
                        console.log(self.instrumentIdentifier);
                    });
                    g_modDebugMgr.AddDebugButton("TemplateID", function() {
                        console.log('TemplateID');
                        console.log(self.templateID);
                    });
                    g_modDebugMgr.AddDebugButton("Source", function() {
                        console.log('Source');
                        console.log(window.document.documentElement.outerHTML);
                    });
                    g_modDebugMgr.AddDebugButton("close", function() {
                        console.log('close');
                        if (self.ingameUi) {
                            console.log('ingameUi');
                            self.ingameUi.closePanel();
                        }
                    });
                    self.initialize();
                } else {
                    setTimeout(() => {
                        self.isDebugEnabled();
                    }, 2000);
                }
            });
        }
    }

    log(msg) {
        const tmp = document.querySelector("#warning_message");
        if (tmp) {
            tmp.innerHTML = tmp.innerHTML + "<p>" + msg + "</p>";
        }
    }

    collapse(collapsed) {
        if (collapsed) {
            this.classList.add("collapsed");
        }
        else {
            this.classList.remove("collapsed");
        }

        this.collapsed = collapsed;
    }

    toggle_collapse() {
        this.collapse(!this.collapsed);
    }

    request_hotkey() {
        const self = this;

        var xhr = new XMLHttpRequest();
        var url = "/hotkey";
        xhr.open("GET", url, true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.onreadystatechange = function () {
            if (xhr.readyState === 4) {
                if (xhr.status === 200) {
                    var json = JSON.parse(xhr.responseText);
                    if (json && json.keycode != null) {
                        self.collapse_hotkey = json.keycode;
                    }
                }
            }
        };
        xhr.send();
    }

    connectedCallback() {
        super.connectedCallback();

        var self = this;

        window.addEventListener("message", (e) => {
            if (self.collapse_hotkey == -1) return;

            try {
                const data = JSON.parse(e.data);
                if (data.type == "KeyboardEvent" && data.data != null) {
                    if (data.data.type == "keydown" && data.data.keyCode == self.collapse_hotkey && data.data.altKey) {
                        self.toggle_collapse();
                    }
                }
            }
            catch (e) {
                /* ignore silently */
            }
        });

        window.addEventListener("keydown", (e) => {
            if (self.collapse_hotkey == -1) return;

            if (e.keyCode == self.collapse_hotkey && e.altKey) {
                self.toggle_collapse();
            }
        });

        setTimeout(() => {
            this.ingameUi = this.querySelector("ingame-ui");

            this.content_iframe = document.getElementById("content_iframe");

            this.warning_message = document.getElementById("warning_message");

            if (this.ingameUi) {
                this.ingameUi.addEventListener("panelActive", (e) => {
                    self.panelActive = true;
                    self.warning_message.classList.add("show");

                    self.request_hotkey();

                    if (self.content_iframe) {
                        self.content_iframe.src = 'http://localhost:9000/index.html';
                    }

                    self.collapse(false);
                });
    
                this.ingameUi.addEventListener("panelInactive", (e) => {
                    self.panelActive = false;
                    self.warning_message.classList.remove("show");
    
                    if (self.content_iframe) {
                        self.content_iframe.src = '';
                    }
                });
            }
        } , 0);
    }

    initialize() {
        if (this.started) {
            return;
        }
        this.started = true;
    }

    disconnectedCallback() {
        super.disconnectedCallback();
    }

    updateImage() {

    }
}

window.customElements.define("ingamepanel-custom", IngamePanelFSKneeboardPanel);
myCheckAutoload();

if (parent && parent.window && parent.window.test_environment) {
    parent.document.addEventListener('testReady', function (e) { 
        console.log("iframe => testReady");
        uis = document.querySelectorAll("ingame-ui");

        for (let ui of uis) {
            console.log("ui", typeof(ui));

            const event = new Event('panelActive');
            setTimeout(() => {
                ui.dispatchEvent(event);
                console.log("Event dispatched...");
            }, 250);
        }
    }, false);
}


