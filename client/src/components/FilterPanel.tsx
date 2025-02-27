import React, { memo, useCallback } from 'react';
import { 
    Box, Typography, Paper, Grid,
    FormControl, InputLabel, Select, MenuItem,
    Slider, Stack, Button
} from '@mui/material';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import { Dayjs } from 'dayjs';
import { debounce } from 'lodash';

export interface FilterOptions {
    startDate: Dayjs | null;
    endDate: Dayjs | null;
    propertyType: string;
    status: string;
    numRooms: [number, number];
    priceRange: [number, number];
    sizeRange: [number, number];
}

interface RangeOptions {
    price: { min: number; max: number };
    size: { min: number; max: number };
    rooms: { min: number; max: number };
}

interface FilterPanelProps {
    pendingFilters: FilterOptions;
    ranges: RangeOptions;
    onFilterChange: (newFilters: FilterOptions) => void;
    onApplyFilters: () => void;
    onResetFilters: () => void;
}

const FilterPanel: React.FC<FilterPanelProps> = ({
    pendingFilters,
    ranges,
    onFilterChange,
    onApplyFilters,
    onResetFilters
}) => {
    // Debounced filter updates for sliders
    const debouncedFilterUpdate = useCallback(
        debounce((newFilters: FilterOptions) => {
            onFilterChange(newFilters);
        }, 100),
        [onFilterChange]
    );

    const handleSliderChange = useCallback((field: keyof FilterOptions, newValue: [number, number]) => {
        const newFilters = {
            ...pendingFilters,
            [field]: newValue
        };
        debouncedFilterUpdate(newFilters);
    }, [pendingFilters, debouncedFilterUpdate]);

    const handleSelectChange = useCallback((field: keyof FilterOptions, value: string) => {
        const newFilters = {
            ...pendingFilters,
            [field]: value
        };
        onFilterChange(newFilters);
    }, [pendingFilters, onFilterChange]);

    const handleDateChange = useCallback((field: 'startDate' | 'endDate', value: Dayjs | null) => {
        const newFilters = {
            ...pendingFilters,
            [field]: value
        };
        onFilterChange(newFilters);
    }, [pendingFilters, onFilterChange]);

    return (
        <Paper sx={{ p: 3, mb: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6">
                    Filters
                </Typography>
                <Box>
                    <Button 
                        variant="outlined" 
                        onClick={onResetFilters} 
                        sx={{ mr: 1 }}
                    >
                        Reset
                    </Button>
                    <Button 
                        variant="contained" 
                        onClick={onApplyFilters}
                    >
                        Apply Filters
                    </Button>
                </Box>
            </Box>
            <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                    <Stack spacing={2}>
                        <DatePicker
                            label="Start Date"
                            value={pendingFilters.startDate}
                            onChange={(newValue) => handleDateChange('startDate', newValue)}
                            slotProps={{ textField: { fullWidth: true } }}
                        />
                        <DatePicker
                            label="End Date"
                            value={pendingFilters.endDate}
                            onChange={(newValue) => handleDateChange('endDate', newValue)}
                            slotProps={{ textField: { fullWidth: true } }}
                        />
                    </Stack>
                </Grid>
                <Grid item xs={12} md={6}>
                    <Stack spacing={2}>
                        <FormControl fullWidth>
                            <InputLabel>Property Type</InputLabel>
                            <Select
                                value={pendingFilters.propertyType}
                                label="Property Type"
                                onChange={(e) => handleSelectChange('propertyType', e.target.value)}
                            >
                                <MenuItem value="all">All</MenuItem>
                                <MenuItem value="appartement">Apartment</MenuItem>
                                <MenuItem value="huis">House</MenuItem>
                            </Select>
                        </FormControl>
                        <FormControl fullWidth>
                            <InputLabel>Status</InputLabel>
                            <Select
                                value={pendingFilters.status}
                                label="Status"
                                onChange={(e) => handleSelectChange('status', e.target.value)}
                            >
                                <MenuItem value="all">All</MenuItem>
                                <MenuItem value="active">Active</MenuItem>
                                <MenuItem value="sold">Sold</MenuItem>
                            </Select>
                        </FormControl>
                    </Stack>
                </Grid>
                <Grid item xs={12} md={4}>
                    <Typography gutterBottom>Number of Rooms</Typography>
                    <Slider
                        value={pendingFilters.numRooms}
                        onChange={(_, newValue) => handleSliderChange('numRooms', newValue as [number, number])}
                        valueLabelDisplay="auto"
                        min={ranges.rooms.min}
                        max={ranges.rooms.max}
                        marks
                    />
                </Grid>
                <Grid item xs={12} md={4}>
                    <Typography gutterBottom>Price Range (€)</Typography>
                    <Slider
                        value={pendingFilters.priceRange}
                        onChange={(_, newValue) => handleSliderChange('priceRange', newValue as [number, number])}
                        valueLabelDisplay="auto"
                        min={ranges.price.min}
                        max={ranges.price.max}
                        step={50000}
                        valueLabelFormat={(value) => `€${(value/1000)}k`}
                    />
                </Grid>
                <Grid item xs={12} md={4}>
                    <Typography gutterBottom>Size Range (m²)</Typography>
                    <Slider
                        value={pendingFilters.sizeRange}
                        onChange={(_, newValue) => handleSliderChange('sizeRange', newValue as [number, number])}
                        valueLabelDisplay="auto"
                        min={ranges.size.min}
                        max={ranges.size.max}
                        step={5}
                        valueLabelFormat={(value) => `${value}m²`}
                    />
                </Grid>
            </Grid>
        </Paper>
    );
};

export default memo(FilterPanel); 