import React, { useState, useEffect } from 'react';
import {
    Box,
    Button,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField,
    Autocomplete,
    Chip,
    Alert,
    FormControl,
    FormHelperText,
    createFilterOptions
} from '@mui/material';
import { MetropolitanArea, MetropolitanAreaFormData } from '../types/metropolitan';
import { api } from '../services/api';

interface MetropolitanAreaFormProps {
    area?: MetropolitanArea | null;
    onSubmit: () => void;
    onCancel: () => void;
}

// Create a filter that allows adding new options
const filter = createFilterOptions<string>();

const MetropolitanAreaForm: React.FC<MetropolitanAreaFormProps> = ({ area, onSubmit, onCancel }) => {
    const [formData, setFormData] = useState<MetropolitanAreaFormData>({
        name: '',
        cities: [],
        center_lat: undefined,
        center_lng: undefined,
        zoom_level: undefined
    });
    const [error, setError] = useState<string | null>(null);
    const [availableCities] = useState<string[]>([
        'Amsterdam', 'Rotterdam', 'Utrecht', 'Den Haag', 'Eindhoven',
        'Amstelveen', 'Diemen', 'Zaandam', 'Haarlem', 'Almere',
        'Schiedam', 'Vlaardingen', 'Capelle aan den IJssel', 'Spijkenisse',
        'Zeist', 'De Bilt', 'Hilversum', 'Maarssen', 'Nieuwegein'
    ]);

    useEffect(() => {
        if (area) {
            setFormData({
                name: area.name,
                cities: area.cities,
                center_lat: area.center_lat,
                center_lng: area.center_lng,
                zoom_level: area.zoom_level
            });
        }
    }, [area]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        // Validate form
        if (!formData.name.trim()) {
            setError('Name is required');
            return;
        }

        if (formData.cities.length === 0) {
            setError('At least one city must be selected');
            return;
        }

        try {
            if (area) {
                await api.updateMetroArea(area.name, formData);
            } else {
                await api.createMetroArea(formData);
            }
            onSubmit();
        } catch (error: any) {
            setError(error.response?.data?.error || 'Failed to save metropolitan area');
        }
    };

    const handleCityChange = (_: any, newValue: (string | string[])[]) => {
        // Handle both string and array inputs
        const processedCities = newValue.map(option => {
            if (typeof option === 'string') {
                // For direct string inputs (freeSolo mode)
                return option.trim();
            }
            return option;
        }).filter(city => city.length > 0); // Filter out empty strings

        setFormData({ ...formData, cities: processedCities as string[] });
        if (error?.includes('city')) {
            setError(null);
        }
    };

    const isCitiesError = error?.includes('city') || (formData.cities.length === 0);

    return (
        <form onSubmit={handleSubmit} noValidate>
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
                        error={error?.includes('Name')}
                    />
                    <FormControl error={isCitiesError} required fullWidth>
                        <Autocomplete
                            multiple
                            freeSolo
                            selectOnFocus
                            clearOnBlur
                            handleHomeEndKeys
                            options={availableCities}
                            value={formData.cities}
                            onChange={handleCityChange}
                            filterOptions={(options, params) => {
                                const filtered = filter(options, params);
                                const { inputValue } = params;
                                
                                // Suggest creating a new value if it's not in the list
                                const isExisting = options.some(
                                    option => inputValue.toLowerCase() === option.toLowerCase()
                                );
                                if (inputValue !== '' && !isExisting) {
                                    filtered.push(inputValue);
                                }

                                return filtered;
                            }}
                            renderTags={(value, getTagProps) =>
                                value.map((option, index) => {
                                    const tagProps = getTagProps({ index });
                                    return (
                                        <Chip
                                            key={index}
                                            label={option}
                                            onDelete={tagProps.onDelete}
                                            disabled={tagProps.disabled}
                                        />
                                    );
                                })
                            }
                            renderInput={(params) => (
                                <TextField
                                    {...params}
                                    label="Cities"
                                    placeholder="Type to add a city"
                                    error={isCitiesError}
                                    required
                                    inputProps={{
                                        ...params.inputProps,
                                        required: formData.cities.length === 0
                                    }}
                                />
                            )}
                        />
                        {isCitiesError && (
                            <FormHelperText>At least one city must be selected</FormHelperText>
                        )}
                    </FormControl>
                    <Box sx={{ display: 'flex', gap: 2 }}>
                        <TextField
                            label="Center Latitude"
                            type="number"
                            value={formData.center_lat || ''}
                            onChange={(e) => setFormData({ ...formData, center_lat: parseFloat(e.target.value) || undefined })}
                            fullWidth
                            inputProps={{ step: 'any' }}
                            disabled={!area} // Only allow editing for existing areas
                        />
                        <TextField
                            label="Center Longitude"
                            type="number"
                            value={formData.center_lng || ''}
                            onChange={(e) => setFormData({ ...formData, center_lng: parseFloat(e.target.value) || undefined })}
                            fullWidth
                            inputProps={{ step: 'any' }}
                            disabled={!area} // Only allow editing for existing areas
                        />
                        <TextField
                            label="Zoom Level"
                            type="number"
                            value={formData.zoom_level || ''}
                            onChange={(e) => setFormData({ ...formData, zoom_level: parseInt(e.target.value) || undefined })}
                            fullWidth
                            inputProps={{ min: 1, max: 20, step: 1 }}
                            disabled={!area} // Only allow editing for existing areas
                        />
                    </Box>
                    {error && !error.includes('city') && (
                        <Alert severity="error" sx={{ mt: 1 }}>
                            {error}
                        </Alert>
                    )}
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onCancel}>Cancel</Button>
                <Button 
                    type="submit" 
                    variant="contained" 
                    color="primary"
                    disabled={formData.cities.length === 0}
                >
                    {area ? 'Update' : 'Create'}
                </Button>
            </DialogActions>
        </form>
    );
};

export default MetropolitanAreaForm; 