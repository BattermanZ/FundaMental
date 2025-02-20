export interface TelegramConfig {
    is_enabled: boolean;
    bot_token: string;
    chat_id: string;
}

export interface TelegramFilters {
    min_price: number | null;
    max_price: number | null;
    min_living_area: number | null;
    max_living_area: number | null;
    min_rooms: number | null;
    max_rooms: number | null;
    districts: string[];
    energy_labels: string[];
}

export const ENERGY_LABELS = ['A++', 'A+', 'A', 'B', 'C', 'D', 'E', 'F', 'G'] as const;
export type EnergyLabel = typeof ENERGY_LABELS[number]; 