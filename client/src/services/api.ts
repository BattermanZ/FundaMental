import axios from 'axios';
import { Property, PropertyStats, AreaStats } from '../types/property';

const API_BASE_URL = 'http://localhost:5250/api';

// Create axios instance with default config
const axiosInstance = axios.create({
    baseURL: API_BASE_URL,
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json',
    }
});

export const api = {
    getAllProperties: async (): Promise<Property[]> => {
        const response = await axiosInstance.get('/properties');
        return response.data;
    },

    getPropertyStats: async (): Promise<PropertyStats> => {
        const response = await axiosInstance.get('/stats');
        return response.data;
    },

    getAreaStats: async (postalPrefix: string): Promise<AreaStats> => {
        const response = await axiosInstance.get(`/areas/${postalPrefix}`);
        return response.data;
    },

    getRecentSales: async (limit: number = 10): Promise<Property[]> => {
        const response = await axiosInstance.get('/recent-sales', {
            params: { limit }
        });
        return response.data;
    },

    // Add a method to trigger geocoding manually if needed
    updateCoordinates: async (): Promise<void> => {
        await axiosInstance.post('/update-coordinates');
    }
}; 