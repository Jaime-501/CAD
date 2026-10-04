export namespace models {
	
	export class Anime {
	    id: string;
	    title: string;
	    cover_url: string;
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new Anime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.cover_url = source["cover_url"];
	        this.provider = source["provider"];
	    }
	}

}

