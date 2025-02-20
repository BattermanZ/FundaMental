export interface MetropolitanArea {
    id: number;
    name: string;
    cities: string[];
    center_lat?: number;
    center_lng?: number;
    zoom_level?: number;
}

export interface MetropolitanCity {
    id: number;
    metropolitan_area_id: number;
    city: string;
    lat?: number;
    lng?: number;
}

export interface MetropolitanAreaFormData {
    name: string;
    cities: string[];
    center_lat?: number;
    center_lng?: number;
    zoom_level?: number;
} 