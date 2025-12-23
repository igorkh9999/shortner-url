export interface Link {
    id?: number;
    short_code: string;
    original_url: string;
    user_id: string;
    created_at: string;
}

export interface Analytics {
    short_code: string;
    total_clicks: number;
    unique_visitors: number;
    clicks_over_time: ClickData[];
    top_referrers: Referrer[];
    click_rate?: number;
    peak_hour?: ClickData;
}

export interface ClickData {
    timestamp: string;
    count: number;
}

export interface Referrer {
    referer: string;
    count: number;
}

export interface LinkStats {
    short_code: string;
    total_clicks: number;
    unique_visitors: number;
}

export interface CreateLinkResponse {
    short_code: string;
    short_url: string;
    original_url: string;
    created_at: string;
}

export interface ListLinksResponse {
    links: LinkInfo[];
}

export interface LinkInfo {
    short_code: string;
    original_url: string;
    created_at: string;
    total_clicks: number;
}
