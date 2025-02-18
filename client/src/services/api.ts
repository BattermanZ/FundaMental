import axios from 'axios';
import { Property, PropertyStats, AreaStats, DateRange } from '../types/property';
import { MetropolitanArea, MetropolitanAreaFormData } from '../types/metropolitan';

// Get the API URL from environment variables, fallback to localhost if not set
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:5250/api';

// Create axios instance with default config
const axiosInstance = axios.create({
    baseURL: API_BASE_URL,
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json',
    }
});

export const api = {
    getAllProperties: async (dateRange: DateRange, metropolitanAreaId?: number | null): Promise<Property[]> => {
        const response = await axiosInstance.get('/properties', {
            params: {
                ...dateRange,
                metropolitanAreaId
            }
        });
        return response.data;
    },

    getPropertyStats: async (dateRange: DateRange, metropolitanAreaId?: number | null): Promise<PropertyStats> => {
        const response = await axiosInstance.get('/properties/stats', {
            params: {
                ...dateRange,
                metropolitanAreaId
            }
        });
        return response.data;
    },

    getAreaStats: async (postalPrefix: string, dateRange: DateRange, metropolitanAreaId?: number | null): Promise<AreaStats> => {
        const response = await axiosInstance.get(`/properties/area/${postalPrefix}`, {
            params: {
                ...dateRange,
                metropolitanAreaId
            }
        });
        return response.data;
    },

    getRecentSales: async (limit: number = 10, dateRange: DateRange, metropolitanAreaId?: number | null): Promise<Property[]> => {
        const response = await axiosInstance.get('/properties/recent', {
            params: { 
                limit,
                ...dateRange,
                metropolitanAreaId
            }
        });
        return response.data;
    },

    // Add a method to trigger geocoding manually if needed
    updateCoordinates: async (): Promise<void> => {
        await axiosInstance.post('/geocode/update');
    },

    // Metropolitan Area endpoints
    getMetropolitanAreas: async (): Promise<MetropolitanArea[]> => {
        const response = await axiosInstance.get('/metropolitan');
        return response.data;
    },

    getMetropolitanArea: async (name: string): Promise<MetropolitanArea> => {
        const response = await axiosInstance.get(`/metropolitan/${name}`);
        return response.data;
    },

    createMetropolitanArea: async (data: MetropolitanAreaFormData): Promise<MetropolitanArea> => {
        const response = await axiosInstance.post('/metropolitan', data);
        return response.data;
    },

    updateMetropolitanArea: async (name: string, data: MetropolitanAreaFormData): Promise<MetropolitanArea> => {
        const response = await axiosInstance.put(`/metropolitan/${name}`, {
            ...data,
            name // Ensure name in body matches URL
        });
        return response.data;
    },

    deleteMetropolitanArea: async (name: string): Promise<void> => {
        await axiosInstance.delete(`/metropolitan/${name}`);
    },

    // Telegram configuration endpoints
    getTelegramConfig: async () => {
        const response = await axiosInstance.get('/telegram/config');
        return response.data;
    },

    updateTelegramConfig: async (config: { bot_token: string; chat_id: string; is_enabled: boolean }) => {
        const response = await axiosInstance.post('/telegram/config', config);
        return response.data;
    },

    testTelegramConfig: async () => {
        const response = await axiosInstance.post('/telegram/config/test');
        return response.data;
    }
}; 