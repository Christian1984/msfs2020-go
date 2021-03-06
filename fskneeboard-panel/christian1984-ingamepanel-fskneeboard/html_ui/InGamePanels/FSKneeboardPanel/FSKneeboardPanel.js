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
                //console.log('Identifier');
                //console.log(self.instrumentIdentifier);
            });
            g_modDebugMgr.AddDebugButton("TemplateID", function() {
                //console.log('TemplateID');
                //console.log(self.templateID);
            });
            g_modDebugMgr.AddDebugButton("Source", function() {
                //console.log('Source');
                //console.log(window.document.documentElement.outerHTML);
            });
            g_modDebugMgr.AddDebugButton("close", function() {
                //console.log('close');
                if (self.ingameUi) {
                    //console.log('ingameUi');
                    self.ingameUi.closePanel();
                }
            });
            this.initialize();
        } else {
            Include.addScript("/JS/debug.js", function () {
                if (typeof g_modDebugMgr != "undefined") {
                    g_modDebugMgr.AddConsole(null);
                    g_modDebugMgr.AddDebugButton("Identifier", function() {
                        //console.log('Identifier');
                        //console.log(self.instrumentIdentifier);
                    });
                    g_modDebugMgr.AddDebugButton("TemplateID", function() {
                        //console.log('TemplateID');
                        //console.log(self.templateID);
                    });
                    g_modDebugMgr.AddDebugButton("Source", function() {
                        //console.log('Source');
                        //console.log(window.document.documentElement.outerHTML);
                    });
                    g_modDebugMgr.AddDebugButton("close", function() {
                        //console.log('close');
                        if (self.ingameUi) {
                            //console.log('ingameUi');
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

    connectedCallback() {
        super.connectedCallback();

        var self = this;

        window.addEventListener("message", (e) => {
            try {
                const data = JSON.parse(e.data);

                if (data.type == "KeyboardEvent" && data.data != null) {
                    if (self.collapse_hotkey == -1) return;

                    if (data.data.type == "keydown" && data.data.keyCode == self.collapse_hotkey && data.data.altKey) {
                        self.toggle_collapse();
                    }
                }
                else if (data.type == "HotkeyConfiguration" && data.data != null && data.data.keyCode != null) {
                    self.collapse_hotkey = data.data.keyCode;
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
        //console.log("iframe => testReady");
        uis = document.querySelectorAll("ingame-ui");

        for (let ui of uis) {
            //console.log("ui", typeof(ui));

            const event = new Event('panelActive');
            setTimeout(() => {
                ui.dispatchEvent(event);
                //console.log("Event dispatched...");
            }, 250);
        }
    }, false);
}


