import { TelegramConfig, TelegramFilters } from '../types/telegram';
import axios from 'axios';

const API_BASE_URL = 'http://localhost:5250/api';

export async function getTelegramConfig(): Promise<TelegramConfig> {
    const response = await axios.get(`${API_BASE_URL}/telegram/config`);
    return response.data;
}

export async function updateTelegramConfig(config: TelegramConfig): Promise<void> {
    await axios.post(`${API_BASE_URL}/telegram/config`, config);
}

export async function testTelegramConfig(): Promise<void> {
    await axios.post(`${API_BASE_URL}/telegram/config/test`);
}

export async function getTelegramFilters(): Promise<TelegramFilters> {
    const response = await axios.get(`${API_BASE_URL}/telegram/filters`);
    return response.data;
}

export async function updateTelegramFilters(filters: TelegramFilters): Promise<void> {
    await axios.post(`${API_BASE_URL}/telegram/filters`, filters);
} 