import React, { useEffect, useState } from 'react';
import { TelegramFilters, ENERGY_LABELS } from '../types/telegram';
import { getTelegramFilters, updateTelegramFilters } from '../api/telegram';
import { toast } from 'react-hot-toast';
import {
    Box,
    Typography,
    Paper,
    TextField,
    Button,
    Stack,
    Grid,
    Chip,
    ToggleButton,
    ToggleButtonGroup
} from '@mui/material';

export default function TelegramFiltersComponent() {
    const [filters, setFilters] = useState<TelegramFilters>({
        min_price: null,
        max_price: null,
        min_living_area: null,
        max_living_area: null,
        min_rooms: null,
        max_rooms: null,
        districts: [],
        energy_labels: [],
    });
    const [loading, setLoading] = useState(false);
    const [newDistrict, setNewDistrict] = useState('');

    useEffect(() => {
        loadFilters();
    }, []);

    const loadFilters = async () => {
        try {
            const data = await getTelegramFilters();
            setFilters({
                ...data,
                districts: data.districts || [],
                energy_labels: data.energy_labels || [],
            });
        } catch (error) {
            toast.error('Failed to load filters');
            // Keep the default empty arrays on error
            setFilters(prev => ({
                ...prev,
                districts: [],
                energy_labels: [],
            }));
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        try {
            await updateTelegramFilters(filters);
            toast.success('Filters updated successfully');
        } catch (error) {
            toast.error('Failed to update filters');
        } finally {
            setLoading(false);
        }
    };

    const handleNumberChange = (field: keyof TelegramFilters, value: string) => {
        const numValue = value === '' ? null : parseInt(value, 10);
        setFilters(prev => ({
            ...prev,
            [field]: numValue,
        }));
    };

    const handleAddDistrict = () => {
        if (newDistrict.length === 4 && /^\d{4}$/.test(newDistrict)) {
            if (!filters.districts.includes(newDistrict)) {
                setFilters(prev => ({
                    ...prev,
                    districts: [...prev.districts, newDistrict],
                }));
            }
            setNewDistrict('');
        } else {
            toast.error('District code must be 4 digits');
        }
    };

    const handleRemoveDistrict = (district: string) => {
        setFilters(prev => ({
            ...prev,
            districts: prev.districts.filter(d => d !== district),
        }));
    };

    const handleEnergyLabelChange = (_: React.MouseEvent<HTMLElement>, newLabels: string[]) => {
        setFilters(prev => ({
            ...prev,
            energy_labels: newLabels,
        }));
    };

    return (
        <Paper sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 3 }}>Notification Filters</Typography>
            <form onSubmit={handleSubmit}>
                <Stack spacing={3}>
                    {/* Price Range */}
                    <Grid container spacing={2}>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                type="number"
                                label="Minimum Price (€)"
                                value={filters.min_price ?? ''}
                                onChange={e => handleNumberChange('min_price', e.target.value)}
                                fullWidth
                                placeholder="No minimum"
                            />
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                type="number"
                                label="Maximum Price (€)"
                                value={filters.max_price ?? ''}
                                onChange={e => handleNumberChange('max_price', e.target.value)}
                                fullWidth
                                placeholder="No maximum"
                            />
                        </Grid>
                    </Grid>

                    {/* Living Area Range */}
                    <Grid container spacing={2}>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                type="number"
                                label="Minimum Living Area (m²)"
                                value={filters.min_living_area ?? ''}
                                onChange={e => handleNumberChange('min_living_area', e.target.value)}
                                fullWidth
                                placeholder="No minimum"
                            />
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                type="number"
                                label="Maximum Living Area (m²)"
                                value={filters.max_living_area ?? ''}
                                onChange={e => handleNumberChange('max_living_area', e.target.value)}
                                fullWidth
                                placeholder="No maximum"
                            />
                        </Grid>
                    </Grid>

                    {/* Number of Rooms Range */}
                    <Grid container spacing={2}>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                type="number"
                                label="Minimum Rooms"
                                value={filters.min_rooms ?? ''}
                                onChange={e => handleNumberChange('min_rooms', e.target.value)}
                                fullWidth
                                placeholder="No minimum"
                            />
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                type="number"
                                label="Maximum Rooms"
                                value={filters.max_rooms ?? ''}
                                onChange={e => handleNumberChange('max_rooms', e.target.value)}
                                fullWidth
                                placeholder="No maximum"
                            />
                        </Grid>
                    </Grid>

                    {/* Districts */}
                    <Box>
                        <Typography variant="subtitle1" sx={{ mb: 1 }}>
                            Districts (4-digit postal code prefix)
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid item xs>
                                <TextField
                                    value={newDistrict}
                                    onChange={e => setNewDistrict(e.target.value)}
                                    fullWidth
                                    placeholder="e.g., 1012"
                                    inputProps={{ maxLength: 4 }}
                                />
                            </Grid>
                            <Grid item>
                                <Button
                                    onClick={handleAddDistrict}
                                    variant="contained"
                                    sx={{ height: '100%' }}
                                >
                                    Add
                                </Button>
                            </Grid>
                        </Grid>
                        <Box sx={{ mt: 2, display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                            {filters.districts.map(district => (
                                <Chip
                                    key={district}
                                    label={district}
                                    onDelete={() => handleRemoveDistrict(district)}
                                    color="primary"
                                />
                            ))}
                        </Box>
                    </Box>

                    {/* Energy Labels */}
                    <Box>
                        <Typography variant="subtitle1" sx={{ mb: 1 }}>
                            Energy Labels
                        </Typography>
                        <ToggleButtonGroup
                            value={filters.energy_labels}
                            onChange={handleEnergyLabelChange}
                            aria-label="energy labels"
                            size="small"
                            color="primary"
                            sx={{ flexWrap: 'wrap' }}
                        >
                            {ENERGY_LABELS.map(label => (
                                <ToggleButton
                                    key={label}
                                    value={label}
                                    aria-label={`energy label ${label}`}
                                >
                                    {label}
                                </ToggleButton>
                            ))}
                        </ToggleButtonGroup>
                    </Box>

                    <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
                        <Button
                            type="submit"
                            disabled={loading}
                            variant="contained"
                        >
                            {loading ? 'Saving...' : 'Save Filters'}
                        </Button>
                    </Box>
                </Stack>
            </form>
        </Paper>
    );
} 