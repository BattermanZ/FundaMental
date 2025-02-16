export interface Property {
    id: number;
    url: string;
    street: string;
    neighborhood: string;
    property_type: string;
    city: string;
    postal_code: string;
    price: number;
    year_built: number;
    living_area: number;
    num_rooms: number;
    status: string;
    listing_date: string;
    selling_date: string;
    scraped_at: string;
    created_at: string;
    latitude: number | null;
    longitude: number | null;
}

export interface PropertyStats {
    total_properties: number;
    average_price: number;
    median_price: number;
    avg_days_to_sell: number;
    total_sold: number;
    price_per_sqm: number;
}

export interface AreaStats {
    postal_code: string;
    property_count: number;
    average_price: number;
    median_price: number;
    avg_price_per_sqm: number;
} 