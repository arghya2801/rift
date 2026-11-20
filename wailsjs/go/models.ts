export namespace git {
	
	export class ChangedFile {
	    status: string;
	    path: string;
	    oldPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new ChangedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.path = source["path"];
	        this.oldPath = source["oldPath"];
	    }
	}
	export class Conflict {
	    markerSize: number;
	    stages: string;
	    wholeOnly?: string;
	    base: string;
	    ours: string;
	    theirs: string;
	    merged: string;
	
	    static createFrom(source: any = {}) {
	        return new Conflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.markerSize = source["markerSize"];
	        this.stages = source["stages"];
	        this.wholeOnly = source["wholeOnly"];
	        this.base = source["base"];
	        this.ours = source["ours"];
	        this.theirs = source["theirs"];
	        this.merged = source["merged"];
	    }
	}
	export class FileStatus {
	    path: string;
	    origPath?: string;
	    index: number;
	    work: number;
	    conflict: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.origPath = source["origPath"];
	        this.index = source["index"];
	        this.work = source["work"];
	        this.conflict = source["conflict"];
	    }
	}
	export class IndexRepair {
	    head: string;
	    paths: string[];
	    index: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new IndexRepair(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.head = source["head"];
	        this.paths = source["paths"];
	        this.index = source["index"];
	        this.message = source["message"];
	    }
	}
	export class Line {
	    from: number;
	    to: number;
	    bottom: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Line(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.bottom = source["bottom"];
	    }
	}
	export class Op {
	    kind: string;
	    conflicts: string[];
	
	    static createFrom(source: any = {}) {
	        return new Op(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.conflicts = source["conflicts"];
	    }
	}
	export class PartialFile {
	    expectedBase?: string;
	    path: string;
	    text?: string;
	    delete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PartialFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedBase = source["expectedBase"];
	        this.path = source["path"];
	        this.text = source["text"];
	        this.delete = source["delete"];
	    }
	}
	export class Ref {
	    name: string;
	    full: string;
	    kind: string;
	    hash: string;
	    upstream: string;
	    track: string;
	    current: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Ref(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.full = source["full"];
	        this.kind = source["kind"];
	        this.hash = source["hash"];
	        this.upstream = source["upstream"];
	        this.track = source["track"];
	        this.current = source["current"];
	    }
	}
	export class Row {
	    hash: string;
	    parents: string[];
	    author: string;
	    email: string;
	    time: number;
	    refs: string[];
	    subject: string;
	    col: number;
	    lines: Line[];
	
	    static createFrom(source: any = {}) {
	        return new Row(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.parents = source["parents"];
	        this.author = source["author"];
	        this.email = source["email"];
	        this.time = source["time"];
	        this.refs = source["refs"];
	        this.subject = source["subject"];
	        this.col = source["col"];
	        this.lines = this.convertValues(source["lines"], Line);
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
	export class Sides {
	    readOnly?: boolean;
	    old: string;
	    new: string;
	    binary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Sides(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.readOnly = source["readOnly"];
	        this.old = source["old"];
	        this.new = source["new"];
	        this.binary = source["binary"];
	    }
	}
	export class Stash {
	    ref: string;
	    subject: string;
	    time: number;
	
	    static createFrom(source: any = {}) {
	        return new Stash(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.subject = source["subject"];
	        this.time = source["time"];
	    }
	}
	export class Status {
	    branch: string;
	    upstream: string;
	    ahead: number;
	    behind: number;
	    files: FileStatus[];
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branch = source["branch"];
	        this.upstream = source["upstream"];
	        this.ahead = source["ahead"];
	        this.behind = source["behind"];
	        this.files = this.convertValues(source["files"], FileStatus);
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
	export class TodoItem {
	    action: string;
	    hash: string;
	    subject: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new TodoItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.hash = source["hash"];
	        this.subject = source["subject"];
	        this.message = source["message"];
	    }
	}
	export class Worktree {
	    path: string;
	    head: string;
	    branch: string;
	    detached: boolean;
	    bare: boolean;
	    locked: boolean;
	    main: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Worktree(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.head = source["head"];
	        this.branch = source["branch"];
	        this.detached = source["detached"];
	        this.bare = source["bare"];
	        this.locked = source["locked"];
	        this.main = source["main"];
	    }
	}

}

export namespace main {
	
	export class Entry {
	    name: string;
	    isDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isDir = source["isDir"];
	    }
	}
	export class Overview {
	    repo: string;
	    wt: git.Worktree;
	    status: git.Status;
	    last: string;
	    op: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Overview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = source["repo"];
	        this.wt = this.convertValues(source["wt"], git.Worktree);
	        this.status = this.convertValues(source["status"], git.Status);
	        this.last = source["last"];
	        this.op = source["op"];
	        this.error = source["error"];
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
	export class Repo {
	    path: string;
	    name: string;
	    worktrees: git.Worktree[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Repo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.worktrees = this.convertValues(source["worktrees"], git.Worktree);
	        this.error = source["error"];
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
	export class Snapshot {
	    status: git.Status;
	    refs: git.Ref[];
	    stashes: git.Stash[];
	    op: git.Op;
	    full: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = this.convertValues(source["status"], git.Status);
	        this.refs = this.convertValues(source["refs"], git.Ref);
	        this.stashes = this.convertValues(source["stashes"], git.Stash);
	        this.op = this.convertValues(source["op"], git.Op);
	        this.full = source["full"];
	        this.error = source["error"];
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

}

