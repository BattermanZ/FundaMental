import React, { useState, useEffect } from 'react';
import {
    Box,
    Button,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField,
    Autocomplete,
    Chip
} from '@mui/material';
import { MetropolitanArea, MetropolitanAreaFormData } from '../types/metropolitan';
import { api } from '../services/api';

interface MetropolitanAreaFormProps {
    area?: MetropolitanArea | null;
    onSubmit: () => void;
    onCancel: () => void;
}

const MetropolitanAreaForm: React.FC<MetropolitanAreaFormProps> = ({ area, onSubmit, onCancel }) => {
    const [formData, setFormData] = useState<MetropolitanAreaFormData>({
        name: '',
        cities: []
    });
    const [availableCities, setAvailableCities] = useState<string[]>([]);

    useEffect(() => {
        if (area) {
            setFormData({
                name: area.name,
                cities: area.cities
            });
        }
        // In a real application, you would fetch the list of available cities from the API
        // For now, we'll use a static list as an example
        setAvailableCities([
            'Amsterdam', 'Rotterdam', 'Utrecht', 'Den Haag', 'Eindhoven',
            'Amstelveen', 'Diemen', 'Zaandam', 'Haarlem', 'Almere',
            'Schiedam', 'Vlaardingen', 'Capelle aan den IJssel', 'Spijkenisse',
            'Zeist', 'De Bilt', 'Hilversum', 'Maarssen', 'Nieuwegein'
        ]);
    }, [area]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            if (area) {
                await api.updateMetropolitanArea(area.name, formData);
            } else {
                await api.createMetropolitanArea(formData);
            }
            onSubmit();
        } catch (error) {
            console.error('Failed to save metropolitan area:', error);
        }
    };

    return (
        <form onSubmit={handleSubmit}>
            <DialogTitle>
                {area ? 'Edit Metropolitan Area' : 'Add Metropolitan Area'}
            </DialogTitle>
            <DialogContent>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
                    <TextField
                        label="Name"
                        value={formData.name}
                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                        required
                        fullWidth
                    />
                    <Autocomplete
                        multiple
                        options={availableCities}
                        value={formData.cities}
                        onChange={(_, newValue) => setFormData({ ...formData, cities: newValue })}
                        renderTags={(value, getTagProps) =>
                            value.map((option, index) => (
                                <Chip
                                    label={option}
                                    {...getTagProps({ index })}
                                />
                            ))
                        }
                        renderInput={(params) => (
                            <TextField
                                {...params}
                                label="Cities"
                                placeholder="Add cities"
                                required
                            />
                        )}
                    />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onCancel}>Cancel</Button>
                <Button type="submit" variant="contained" color="primary">
                    {area ? 'Update' : 'Create'}
                </Button>
            </DialogActions>
        </form>
    );
};

export default MetropolitanAreaForm; 