export namespace main {
	
	export class DetalleResultado {
	    detail: models.AnimeDetail;
	    episodes: models.Episode[];
	
	    static createFrom(source: any = {}) {
	        return new DetalleResultado(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detail = this.convertValues(source["detail"], models.AnimeDetail);
	        this.episodes = this.convertValues(source["episodes"], models.Episode);
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
	export class AnimeDetail {
	    id: string;
	    title: string;
	    provider: string;
	    type: string;
	    status: string;
	    genres: string[];
	    synopsis: string;
	    cover: string;
	    total_episodes: number;
	
	    static createFrom(source: any = {}) {
	        return new AnimeDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.provider = source["provider"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.genres = source["genres"];
	        this.synopsis = source["synopsis"];
	        this.cover = source["cover"];
	        this.total_episodes = source["total_episodes"];
	    }
	}
	export class Episode {
	    id: string;
	    number: number;
	    title: string;
	    thumbnail: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new Episode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.thumbnail = source["thumbnail"];
	        this.kind = source["kind"];
	    }
	}
	export class Subtitle {
	    url: string;
	    lang: string;
	
	    static createFrom(source: any = {}) {
	        return new Subtitle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.lang = source["lang"];
	    }
	}
	export class VideoSource {
	    url: string;
	    type: string;
	    quality: string;
	    server: string;
	    headers: Record<string, string>;
	    referer: string;
	    subtitles?: Subtitle[];
	
	    static createFrom(source: any = {}) {
	        return new VideoSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.type = source["type"];
	        this.quality = source["quality"];
	        this.server = source["server"];
	        this.headers = source["headers"];
	        this.referer = source["referer"];
	        this.subtitles = this.convertValues(source["subtitles"], Subtitle);
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
	export class WatchProgress {
	    anime_id: string;
	    episode_id: string;
	    current_time: number;
	    total_duration: number;
	    is_watched: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WatchProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anime_id = source["anime_id"];
	        this.episode_id = source["episode_id"];
	        this.current_time = source["current_time"];
	        this.total_duration = source["total_duration"];
	        this.is_watched = source["is_watched"];
	    }
	}

}

