export namespace core {
	
	export class GifOptions {
	    DelayMS: number;
	
	    static createFrom(source: any = {}) {
	        return new GifOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DelayMS = source["DelayMS"];
	    }
	}

}

export namespace main {
	
	export class GenerateResult {
	    OK: boolean;
	    Message: string;
	    FrameCount: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.OK = source["OK"];
	        this.Message = source["Message"];
	        this.FrameCount = source["FrameCount"];
	    }
	}

}

export namespace session {
	
	export class FrameItem {
	    ID: string;
	    Path: string;
	    Name: string;
	    ThumbDataURL: string;
	    Width: number;
	    Height: number;
	    Format: string;
	
	    static createFrom(source: any = {}) {
	        return new FrameItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Path = source["Path"];
	        this.Name = source["Name"];
	        this.ThumbDataURL = source["ThumbDataURL"];
	        this.Width = source["Width"];
	        this.Height = source["Height"];
	        this.Format = source["Format"];
	    }
	}
	export class AddResult {
	    Frames: FrameItem[];
	    Added: number;
	    Skipped: number;
	    Message: string;
	
	    static createFrom(source: any = {}) {
	        return new AddResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Frames = this.convertValues(source["Frames"], FrameItem);
	        this.Added = source["Added"];
	        this.Skipped = source["Skipped"];
	        this.Message = source["Message"];
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

