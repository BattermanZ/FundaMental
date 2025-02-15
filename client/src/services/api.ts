import axios from 'axios';
import { Property, PropertyStats, AreaStats } from '../types/property';

const API_BASE_URL = 'http://localhost:5250/api';

export const api = {
    getAllProperties: async (): Promise<Property[]> => {
        const response = await axios.get(`${API_BASE_URL}/properties`);
        return response.data;
    },

    getPropertyStats: async (): Promise<PropertyStats> => {
        const response = await axios.get(`${API_BASE_URL}/stats`);
        return response.data;
    },

    getAreaStats: async (postalPrefix: string): Promise<AreaStats> => {
        const response = await axios.get(`${API_BASE_URL}/areas/${postalPrefix}`);
        return response.data;
    },

    getRecentSales: async (limit: number = 10): Promise<Property[]> => {
        const response = await axios.get(`${API_BASE_URL}/recent-sales`, {
            params: { limit }
        });
        return response.data;
    }
}; 