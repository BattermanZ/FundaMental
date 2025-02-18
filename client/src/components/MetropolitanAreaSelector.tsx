import React, { useState, useEffect } from 'react';
import {
    FormControl,
    InputLabel,
    MenuItem,
    Select,
    SelectChangeEvent,
} from '@mui/material';
import { MetropolitanArea } from '../types/metropolitan';
import { api } from '../services/api';

interface MetropolitanAreaSelectorProps {
    value: number | null;
    onChange: (areaId: number | null) => void;
}

const MetropolitanAreaSelector: React.FC<MetropolitanAreaSelectorProps> = ({ value, onChange }) => {
    const [areas, setAreas] = useState<MetropolitanArea[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchAreas = async () => {
            try {
                const data = await api.getMetropolitanAreas();
                setAreas(data || []);
            } catch (error) {
                console.error('Failed to fetch metropolitan areas:', error);
                setAreas([]);
            } finally {
                setLoading(false);
            }
        };
        fetchAreas();
    }, []);

    const handleChange = (event: SelectChangeEvent<number | ''>) => {
        const newValue = event.target.value;
        onChange(newValue === '' ? null : Number(newValue));
    };

    return (
        <FormControl fullWidth size="small" sx={{ bgcolor: 'white', borderRadius: 1 }}>
            <InputLabel id="metro-area-select-label">Metropolitan Area</InputLabel>
            <Select
                labelId="metro-area-select-label"
                id="metro-area-select"
                value={value ?? ''}
                label="Metropolitan Area"
                onChange={handleChange}
                disabled={loading}
            >
                <MenuItem value="">
                    <em>All Areas</em>
                </MenuItem>
                {areas.map((area) => (
                    <MenuItem key={area.id} value={area.id}>
                        {area.name}
                    </MenuItem>
                ))}
            </Select>
        </FormControl>
    );
};

export default MetropolitanAreaSelector; 