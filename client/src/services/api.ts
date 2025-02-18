import axios from 'axios';
import { Property, PropertyStats, AreaStats, DateRange } from '../types/property';

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
    getAllProperties: async (dateRange: DateRange): Promise<Property[]> => {
        const response = await axiosInstance.get('/properties', {
            params: dateRange
        });
        return response.data;
    },

    getPropertyStats: async (dateRange: DateRange): Promise<PropertyStats> => {
        const response = await axiosInstance.get('/properties/stats', {
            params: dateRange
        });
        return response.data;
    },

    getAreaStats: async (postalPrefix: string, dateRange: DateRange): Promise<AreaStats> => {
        const response = await axiosInstance.get(`/properties/area/${postalPrefix}`, {
            params: dateRange
        });
        return response.data;
    },

    getRecentSales: async (limit: number = 10, dateRange: DateRange): Promise<Property[]> => {
        const response = await axiosInstance.get('/properties/recent', {
            params: { 
                limit,
                ...dateRange
            }
        });
        return response.data;
    },

    // Add a method to trigger geocoding manually if needed
    updateCoordinates: async (): Promise<void> => {
        await axiosInstance.post('/geocode/update');
    }
}; 