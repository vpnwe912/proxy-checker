export namespace main {
	
	export class ThreeProxySettings {
	    exePath: string;
	    workDir: string;
	    internalIp: string;
	    proxyPort: string;
	    adminPort: string;
	    useDaemon: boolean;
	    authIpOnly: boolean;
	    allowedIp: string;
	    parentType: string;
	    timeoutLine: string;
	
	    static createFrom(source: any = {}) {
	        return new ThreeProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exePath = source["exePath"];
	        this.workDir = source["workDir"];
	        this.internalIp = source["internalIp"];
	        this.proxyPort = source["proxyPort"];
	        this.adminPort = source["adminPort"];
	        this.useDaemon = source["useDaemon"];
	        this.authIpOnly = source["authIpOnly"];
	        this.allowedIp = source["allowedIp"];
	        this.parentType = source["parentType"];
	        this.timeoutLine = source["timeoutLine"];
	    }
	}
	export class AppConfig {
	    listenAddr: string;
	    proxyApiUrl: string;
	    proxyTypeMode: string;
	    autoImport: boolean;
	    autoImportSec: number;
	    autoImportUnit: string;
	    testUrl: string;
	    checkTimeoutSec: number;
	    monitorEverySec: number;
	    workers: number;
	    allowInsecure: boolean;
	    useCurl: boolean;
	    threeProxy: ThreeProxySettings;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listenAddr = source["listenAddr"];
	        this.proxyApiUrl = source["proxyApiUrl"];
	        this.proxyTypeMode = source["proxyTypeMode"];
	        this.autoImport = source["autoImport"];
	        this.autoImportSec = source["autoImportSec"];
	        this.autoImportUnit = source["autoImportUnit"];
	        this.testUrl = source["testUrl"];
	        this.checkTimeoutSec = source["checkTimeoutSec"];
	        this.monitorEverySec = source["monitorEverySec"];
	        this.workers = source["workers"];
	        this.allowInsecure = source["allowInsecure"];
	        this.useCurl = source["useCurl"];
	        this.threeProxy = this.convertValues(source["threeProxy"], ThreeProxySettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Proxy {
	    host: string;
	    port: string;
	    login?: string;
	    password?: string;
	    type: string;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new Proxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.login = source["login"];
	        this.password = source["password"];
	        this.type = source["type"];
	        this.source = source["source"];
	    }
	}
	export class CheckResult {
	    proxy: Proxy;
	    ok: boolean;
	    statusCode: number;
	    error?: string;
	    duration: number;
	    checkedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxy = this.convertValues(source["proxy"], Proxy);
	        this.ok = source["ok"];
	        this.statusCode = source["statusCode"];
	        this.error = source["error"];
	        this.duration = source["duration"];
	        this.checkedAt = source["checkedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class StateSnapshot {
	    config: AppConfig;
	    proxies: Proxy[];
	    results: CheckResult[];
	    logs: string[];
	    activeProxy?: Proxy;
	    monitorRunning: boolean;
	    importRunning: boolean;
	    threeProxyRun: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StateSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], AppConfig);
	        this.proxies = this.convertValues(source["proxies"], Proxy);
	        this.results = this.convertValues(source["results"], CheckResult);
	        this.logs = source["logs"];
	        this.activeProxy = this.convertValues(source["activeProxy"], Proxy);
	        this.monitorRunning = source["monitorRunning"];
	        this.importRunning = source["importRunning"];
	        this.threeProxyRun = source["threeProxyRun"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class apiResponse {
	    ok: boolean;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}

}

