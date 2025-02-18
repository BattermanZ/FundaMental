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
    FormHelperText
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
                cities: area.cities
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
                await api.updateMetropolitanArea(area.name, formData);
            } else {
                await api.createMetropolitanArea(formData);
            }
            onSubmit();
        } catch (error: any) {
            setError(error.response?.data?.error || 'Failed to save metropolitan area');
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
                            options={availableCities}
                            value={formData.cities}
                            onChange={(_, newValue) => {
                                setFormData({ ...formData, cities: newValue });
                                if (error?.includes('city')) {
                                    setError(null);
                                }
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