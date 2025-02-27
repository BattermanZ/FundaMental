import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { Property } from '../types/property';
import { api } from '../services/api';
import {
    Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
    BarChart, Bar, ScatterChart, Scatter, ResponsiveContainer,
    ComposedChart, Area, Label
} from 'recharts';
import { 
    Box, Typography, CircularProgress, Paper, Grid,
    FormControl, InputLabel, Select, MenuItem,
    Slider, Stack, Button
} from '@mui/material';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import dayjs, { Dayjs } from 'dayjs';
import * as d3 from 'd3';
import PriceHeatmap from './PriceHeatmap';

interface FilterOptions {
    startDate: Dayjs | null;
    endDate: Dayjs | null;
    propertyType: string;
    status: string;
    numRooms: [number, number];
    priceRange: [number, number];
    sizeRange: [number, number];
}

interface PropertyChartsProps {
    metropolitanAreaId?: number | null;
}

const PropertyCharts: React.FC<PropertyChartsProps> = ({ metropolitanAreaId }) => {
    const [properties, setProperties] = useState<Property[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    
    // Separate current filters and pending filters
    const [filters, setFilters] = useState<FilterOptions>({
        startDate: null,
        endDate: null,
        propertyType: 'all',
        status: 'all',
        numRooms: [1, 10],
        priceRange: [0, 2000000],
        sizeRange: [0, 300]
    });
    
    const [pendingFilters, setPendingFilters] = useState<FilterOptions>(filters);
    
    // Range limits for sliders
    const [ranges, setRanges] = useState({
        price: { min: 0, max: 2000000 },
        size: { min: 0, max: 300 },
        rooms: { min: 1, max: 10 }
    });

    // Fetch data
    useEffect(() => {
        const fetchData = async () => {
            try {
                setLoading(true);
                const data = await api.getAllProperties({
                    startDate: undefined,
                    endDate: undefined
                }, metropolitanAreaId);
                setProperties(data);
                
                // Calculate actual ranges from data
                const prices = data.map(p => p.price).filter(Boolean);
                const sizes = data.map(p => p.living_area).filter(Boolean);
                const rooms = data.map(p => p.num_rooms).filter(Boolean);
                
                const newRanges = {
                    price: {
                        min: Math.min(...prices),
                        max: Math.max(...prices)
                    },
                    size: {
                        min: Math.min(...sizes),
                        max: Math.max(...sizes)
                    },
                    rooms: {
                        min: Math.min(...rooms),
                        max: Math.max(...rooms)
                    }
                };
                
                setRanges(newRanges);
                
                // Initialize filters with actual ranges
                const initialFilters: FilterOptions = {
                    startDate: null,
                    endDate: null,
                    propertyType: 'all',
                    status: 'all',
                    numRooms: [newRanges.rooms.min, newRanges.rooms.max] as [number, number],
                    priceRange: [newRanges.price.min, newRanges.price.max] as [number, number],
                    sizeRange: [newRanges.size.min, newRanges.size.max] as [number, number]
                };
                
                setFilters(initialFilters);
                setPendingFilters(initialFilters);
                
                setError(null);
            } catch (error) {
                console.error('Failed to fetch data:', error);
                setError('Failed to load property data');
            } finally {
                setLoading(false);
            }
        };

        fetchData();
    }, [metropolitanAreaId]);

    // Memoize filtered properties
    const filteredPropertiesMemo = useMemo(() => {
        if (!properties || properties.length === 0) return [];
        return properties.filter(property => {
            // Date filter - check listing_date/scraped_at for active and selling_date for sold
            if (filters.startDate) {
                const effectiveDate = property.status === 'active' 
                    ? (property.listing_date ? dayjs(property.listing_date) : dayjs(property.scraped_at))
                    : (property.selling_date ? dayjs(property.selling_date) : null);
                
                if (effectiveDate && effectiveDate.isBefore(filters.startDate)) {
                    return false;
                }
            }
            
            if (filters.endDate) {
                const effectiveDate = property.status === 'active' 
                    ? (property.listing_date ? dayjs(property.listing_date) : dayjs(property.scraped_at))
                    : (property.selling_date ? dayjs(property.selling_date) : null);
                
                if (effectiveDate && effectiveDate.isAfter(filters.endDate)) {
                    return false;
                }
            }
            
            if (filters.propertyType !== 'all' && property.property_type !== filters.propertyType) return false;
            if (filters.status !== 'all' && property.status !== filters.status) return false;
            if (property.num_rooms && (
                property.num_rooms < filters.numRooms[0] ||
                property.num_rooms > filters.numRooms[1]
            )) return false;
            if (property.price && (
                property.price < filters.priceRange[0] ||
                property.price > filters.priceRange[1]
            )) return false;
            if (property.living_area && (
                property.living_area < filters.sizeRange[0] ||
                property.living_area > filters.sizeRange[1]
            )) return false;
            
            return true;
        });
    }, [properties, filters]);

    // Memoize chart data
    const scatterData = useMemo(() => {
        return filteredPropertiesMemo
            .filter(p => p.living_area && p.price)
            .map(p => ({
                living_area: p.living_area,
                price: p.price,
                postal_code: p.postal_code.substring(0, 4),
                num_rooms: p.num_rooms || 1,
                price_per_sqm: p.price / p.living_area
            }));
    }, [filteredPropertiesMemo]);

    const priceByPostalCodeData = useMemo(() => {
        const postalGroups = d3.group(
            filteredPropertiesMemo.filter(p => p.price),
            d => d.postal_code.substring(0, 4)
        );

        return Array.from(postalGroups, ([postal_code, group]) => ({
            postal_code,
            avg_price: d3.mean(group, d => d.price) || 0,
            median_price: d3.median(group, d => d.price) || 0,
            count: group.length
        })).sort((a, b) => b.avg_price - a.avg_price);
    }, [filteredPropertiesMemo]);

    const timeSeriesData = useMemo(() => {
        // Separate active and sold properties
        const soldProperties = filteredPropertiesMemo
            .filter(p => p.status === 'sold' && p.selling_date && dayjs(p.selling_date).isValid() && dayjs(p.selling_date).year() >= 2024)
            .sort((a, b) => dayjs(a.selling_date).valueOf() - dayjs(b.selling_date).valueOf());

        const activeProperties = filteredPropertiesMemo
            .filter(p => 
                p.status === 'active' && 
                p.scraped_at && 
                dayjs(p.scraped_at).isValid() && 
                dayjs(p.scraped_at).year() >= 2024
            )
            .map(p => ({
                ...p,
                effectiveDate: p.scraped_at
            }))
            .sort((a, b) => dayjs(a.effectiveDate).valueOf() - dayjs(b.effectiveDate).valueOf());

        if (soldProperties.length === 0 && activeProperties.length === 0) return [];

        // Group properties by month
        const soldMonthlyGroups = d3.group(soldProperties, d => 
            dayjs(d.selling_date).format('YYYY-MM')
        );

        const activeMonthlyGroups = d3.group(activeProperties, d => 
            dayjs(d.effectiveDate).format('YYYY-MM')
        );

        // Get unique months from both groups
        const uniqueMonths = new Set([
            ...Array.from(soldMonthlyGroups.keys()),
            ...Array.from(activeMonthlyGroups.keys())
        ]);

        // Convert to array and sort chronologically
        const sortedMonths = Array.from(uniqueMonths).sort();

        // Create data points only for months that have data
        return sortedMonths.map(month => {
            const soldGroup = soldMonthlyGroups.get(month) || [];
            const activeGroup = activeMonthlyGroups.get(month) || [];
            const allProperties = [...soldGroup, ...activeGroup];

            return {
                month,
                avg_price: d3.mean(allProperties, d => d.price) || 0,
                median_price: d3.median(allProperties, d => d.price) || 0,
                avg_days_to_sell: d3.mean(soldGroup, d => {
                    if (!d.listing_date || !d.selling_date) return null;
                    const listDate = dayjs(d.listing_date);
                    const sellDate = dayjs(d.selling_date);
                    if (!listDate.isValid() || !sellDate.isValid()) return null;
                    return sellDate.diff(listDate, 'day');
                }) || 0,
                count: allProperties.length,
                active_count: activeGroup.length,
                sold_count: soldGroup.length
            };
        });
    }, [filteredPropertiesMemo]);

    const pricePerSqmData = useMemo(() => {
        const postalGroups = d3.group(
            filteredPropertiesMemo.filter(p => p.living_area && p.price),
            d => d.postal_code.substring(0, 4)
        );

        return Array.from(postalGroups, ([postal_code, group]) => ({
            postal_code,
            avg_price_per_sqm: d3.mean(group, d => d.price / d.living_area) || 0,
            median_price_per_sqm: d3.median(group, d => d.price / d.living_area) || 0,
            count: group.length
        })).sort((a, b) => b.avg_price_per_sqm - a.avg_price_per_sqm);
    }, [filteredPropertiesMemo]);

    // Rooms Impact Analysis Data
    const roomsImpactData = useMemo(() => {
        const roomGroups = d3.group(
            filteredPropertiesMemo.filter(p => p.num_rooms && p.price),
            d => d.num_rooms
        );

        return Array.from(roomGroups, ([rooms, group]) => ({
            rooms: Number(rooms),
            count: group.length,
            avgPrice: d3.mean(group, d => d.price) || 0,
            medianPrice: d3.median(group, d => d.price) || 0,
            avgSize: d3.mean(group, d => d.living_area) || 0
        })).sort((a, b) => a.rooms - b.rooms);
    }, [filteredPropertiesMemo]);

    // Room Price Premium Analysis Data
    const roomPricePremiumData = useMemo(() => {
        const roomGroups = d3.group(
            filteredPropertiesMemo.filter(p => p.num_rooms && p.price),
            d => d.num_rooms
        );

        const data = Array.from(roomGroups, ([rooms, group]) => ({
            rooms: Number(rooms),
            medianPrice: d3.median(group, d => d.price) || 0,
            count: group.length
        })).sort((a, b) => a.rooms - b.rooms);

        // Calculate price premium compared to previous room count
        return data.map((item, index) => ({
            rooms: item.rooms,
            count: item.count,
            medianPrice: item.medianPrice,
            pricePremium: index > 0 ? item.medianPrice - data[index - 1].medianPrice : 0,
            percentageIncrease: index > 0 ? ((item.medianPrice - data[index - 1].medianPrice) / data[index - 1].medianPrice) * 100 : 0
        })).filter(d => d.rooms <= 10);  // Filter to show only up to 10 rooms
    }, [filteredPropertiesMemo]);

    // Calculate regression line for scatter plot
    const calculateRegressionLine = useCallback((data: any[]) => {
        const xValues = data.map(d => d.living_area);
        const yValues = data.map(d => d.price);
        
        const xMean = d3.mean(xValues) || 0;
        const yMean = d3.mean(yValues) || 0;
        
        const ssXX = d3.sum(xValues, x => Math.pow(x - xMean, 2));
        const ssXY = d3.sum(data, d => (d.living_area - xMean) * (d.price - yMean));
        
        const slope = ssXY / ssXX;
        const intercept = yMean - slope * xMean;
        
        const minX = Math.min(...xValues);
        const maxX = Math.max(...xValues);
        
        return [
            { x: minX, y: slope * minX + intercept },
            { x: maxX, y: slope * maxX + intercept }
        ];
    }, []);

    const regressionLine = useMemo(() => {
        return calculateRegressionLine(scatterData);
    }, [scatterData, calculateRegressionLine]);

    // Memoize handlers
    const applyFilters = useCallback(() => {
        setFilters(pendingFilters);
    }, [pendingFilters]);

    const resetFilters = useCallback(() => {
        const resetValues: FilterOptions = {
            startDate: null,
            endDate: null,
            propertyType: 'all',
            status: 'all',
            numRooms: [ranges.rooms.min, ranges.rooms.max] as [number, number],
            priceRange: [ranges.price.min, ranges.price.max] as [number, number],
            sizeRange: [ranges.size.min, ranges.size.max] as [number, number]
        };
        setPendingFilters(resetValues);
        setFilters(resetValues);
    }, [ranges]);

    // Memoize FilterPanel component
    const FilterPanel = useMemo(() => {
        return (
            <Paper sx={{ p: 3, mb: 3 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                    <Typography variant="h6">
                        Filters
                    </Typography>
                    <Box>
                        <Button 
                            variant="outlined" 
                            onClick={resetFilters} 
                            sx={{ mr: 1 }}
                        >
                            Reset
                        </Button>
                        <Button 
                            variant="contained" 
                            onClick={applyFilters}
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
                                onChange={(newValue) => setPendingFilters(prev => ({ ...prev, startDate: newValue }))}
                                slotProps={{ textField: { fullWidth: true } }}
                            />
                            <DatePicker
                                label="End Date"
                                value={pendingFilters.endDate}
                                onChange={(newValue) => setPendingFilters(prev => ({ ...prev, endDate: newValue }))}
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
                                    onChange={(e) => setPendingFilters(prev => ({ ...prev, propertyType: e.target.value }))}
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
                                    onChange={(e) => setPendingFilters(prev => ({ ...prev, status: e.target.value }))}
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
                            onChange={(_, newValue) => setPendingFilters(prev => ({ ...prev, numRooms: newValue as [number, number] }))}
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
                            onChange={(_, newValue) => setPendingFilters(prev => ({ ...prev, priceRange: newValue as [number, number] }))}
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
                            onChange={(_, newValue) => setPendingFilters(prev => ({ ...prev, sizeRange: newValue as [number, number] }))}
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
    }, [pendingFilters, ranges, applyFilters, resetFilters]);

    if (loading) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="400px">
                <CircularProgress />
            </Box>
        );
    }

    if (error) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="400px">
                <Typography color="error">{error}</Typography>
            </Box>
        );
    }

    return (
        <Box mt={4}>
            {FilterPanel}
            <Grid container spacing={3}>
                {/* Price Heatmap */}
                <Grid item xs={12}>
                    <PriceHeatmap properties={filteredPropertiesMemo} />
                </Grid>

                {/* Price vs Living Area Scatter Plot */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price vs Living Area
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <ScatterChart margin={{ top: 20, right: 20, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis 
                                    dataKey="living_area" 
                                    name="Living Area" 
                                    unit="m²"
                                    type="number"
                                >
                                    <Label value="Living Area (m²)" offset={-10} position="insideBottom" />
                                </XAxis>
                                <YAxis 
                                    dataKey="price" 
                                    name="Price" 
                                    unit="€"
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Price (€)" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any, name: string) => {
                                        if (name === 'Price') return `€${Number(value).toLocaleString()}`;
                                        if (name === 'Living Area') return `${value} m²`;
                                        return value;
                                    }}
                                />
                                <Legend />
                                <Scatter 
                                    name="Properties" 
                                    data={scatterData} 
                                    fill="#8884d8"
                                />
                                <Line
                                    name="Regression Line"
                                    data={regressionLine}
                                    dataKey="y"
                                    stroke="#ff7300"
                                    dot={false}
                                />
                            </ScatterChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Price by Postal Code */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price by Postal Code
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <BarChart data={priceByPostalCodeData} margin={{ top: 20, right: 20, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="postal_code" />
                                <YAxis 
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Price (€)" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any) => `€${Number(value).toLocaleString()}`}
                                />
                                <Legend />
                                <Bar 
                                    dataKey="avg_price" 
                                    fill="#8884d8" 
                                    name="Average Price"
                                />
                                <Bar 
                                    dataKey="median_price" 
                                    fill="#82ca9d" 
                                    name="Median Price"
                                />
                            </BarChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Time Series */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price Trends Over Time
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <ComposedChart data={timeSeriesData} margin={{ top: 20, right: 60, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis 
                                    dataKey="month" 
                                    tickFormatter={(value) => {
                                        // Parse YYYY-MM format
                                        const [year, month] = value.split('-');
                                        return dayjs(`${year}-${month}-01`).format('MMM YY');
                                    }}
                                />
                                <YAxis 
                                    yAxisId="left"
                                    orientation="left"
                                    tickFormatter={(value) => value.toFixed(0)}
                                >
                                    <Label value="Number of Properties" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <YAxis 
                                    yAxisId="right"
                                    orientation="right"
                                    tickFormatter={(value) => value.toFixed(0)}
                                >
                                    <Label value="Days to Sell" angle={90} position="insideRight" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any, name: string) => {
                                        if (name === 'avg_days_to_sell') return [value.toFixed(1), 'Avg Days to Sell'];
                                        if (name === 'active_count') return [value, 'Active Properties'];
                                        if (name === 'sold_count') return [value, 'Sold Properties'];
                                        return [value, name];
                                    }}
                                    labelFormatter={(label) => dayjs(label).format('MMMM YYYY')}
                                />
                                <Legend />
                                <Line
                                    yAxisId="right"
                                    type="monotone"
                                    dataKey="avg_days_to_sell"
                                    stroke="#ffc658"
                                    name="Avg Days to Sell"
                                />
                                <Area
                                    yAxisId="left"
                                    type="monotone"
                                    dataKey="active_count"
                                    fill="#82ca9d"
                                    stroke="#82ca9d"
                                    opacity={0.1}
                                    name="Active Properties"
                                    stackId="1"
                                />
                                <Area
                                    yAxisId="left"
                                    type="monotone"
                                    dataKey="sold_count"
                                    fill="#8884d8"
                                    stroke="#8884d8"
                                    opacity={0.1}
                                    name="Sold Properties"
                                    stackId="1"
                                />
                            </ComposedChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Price per Square Meter Analysis */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price per Square Meter by Postal Code
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <BarChart data={pricePerSqmData} margin={{ top: 20, right: 20, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="postal_code" />
                                <YAxis 
                                    tickFormatter={(value) => `€${value.toFixed(0)}`}
                                >
                                    <Label value="Price per m² (€)" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any) => `€${Number(value).toFixed(0)}/m²`}
                                />
                                <Legend />
                                <Bar 
                                    dataKey="avg_price_per_sqm" 
                                    fill="#8884d8" 
                                    name="Average Price/m²"
                                />
                                <Bar 
                                    dataKey="median_price_per_sqm" 
                                    fill="#82ca9d" 
                                    name="Median Price/m²"
                                />
                            </BarChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Market Velocity Dashboard */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Market Velocity Analysis
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <ComposedChart data={timeSeriesData} margin={{ top: 20, right: 60, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis 
                                    dataKey="month" 
                                    tickFormatter={(value) => {
                                        // Parse YYYY-MM format
                                        const [year, month] = value.split('-');
                                        return dayjs(`${year}-${month}-01`).format('MMM YY');
                                    }}
                                />
                                <YAxis 
                                    yAxisId="left"
                                    orientation="left"
                                    tickFormatter={(value) => value.toFixed(0)}
                                >
                                    <Label value="Days to Sell" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <YAxis 
                                    yAxisId="right"
                                    orientation="right"
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Price Range (€)" angle={90} position="insideRight" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any, name: string) => {
                                        if (name.includes('Price')) return `€${Number(value).toLocaleString()}`;
                                        return value.toFixed(1);
                                    }}
                                    labelFormatter={(label) => dayjs(label).format('MMMM YYYY')}
                                />
                                <Legend />
                                <Bar
                                    yAxisId="left"
                                    dataKey="avg_days_to_sell"
                                    fill="#8884d8"
                                    name="Average Days to Sell"
                                />
                                <Line
                                    yAxisId="right"
                                    type="monotone"
                                    dataKey="avg_price"
                                    stroke="#82ca9d"
                                    name="Average Price"
                                />
                                <Line
                                    yAxisId="right"
                                    type="monotone"
                                    dataKey="median_price"
                                    stroke="#ffc658"
                                    name="Median Price"
                                />
                            </ComposedChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Room Distribution Analysis */}
                <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price Premium per Additional Room
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <ComposedChart 
                                data={roomPricePremiumData} 
                                margin={{ top: 20, right: 60, bottom: 20, left: 60 }}
                            >
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis 
                                    dataKey="rooms"
                                    label={{ value: 'Number of Rooms', position: 'insideBottom', offset: -10 }}
                                />
                                <YAxis 
                                    yAxisId="left"
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Price Premium (€)" angle={-90} position="center" offset={0} dx={-50} />
                                </YAxis>
                                <YAxis 
                                    yAxisId="right"
                                    orientation="right"
                                    tickFormatter={(value) => `${value.toFixed(1)}%`}
                                >
                                    <Label value="Percentage Increase" angle={90} position="center" offset={0} dx={50} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any, name: string, props: any) => {
                                        if (name === "Price Premium") {
                                            const roundedValue = Math.round(value / 1000) * 1000;
                                            return [`€${Number(roundedValue).toLocaleString()} (${props.payload.count} properties)`, name];
                                        }
                                        if (name === "Percentage Increase") {
                                            return [`${value.toFixed(1)}% (${props.payload.count} properties)`, name];
                                        }
                                        return [value, name];
                                    }}
                                />
                                <Legend 
                                    verticalAlign="bottom"
                                    align="center"
                                    layout="horizontal"
                                    wrapperStyle={{
                                        paddingTop: "20px"
                                    }}
                                />
                                <Bar 
                                    yAxisId="left"
                                    dataKey="pricePremium" 
                                    fill="#8884d8" 
                                    name="Price Premium"
                                />
                                <Line 
                                    yAxisId="right"
                                    type="monotone"
                                    dataKey="percentageIncrease" 
                                    stroke="#82ca9d" 
                                    name="Percentage Increase"
                                />
                            </ComposedChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Property Features Impact */}
                <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price Impact by Number of Rooms
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <BarChart 
                                data={roomsImpactData.filter(d => d.rooms <= 10)} 
                                margin={{ top: 20, right: 30, bottom: 20, left: 60 }}
                            >
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis 
                                    dataKey="rooms"
                                    label={{ value: 'Number of Rooms', position: 'insideBottom', offset: -10 }}
                                />
                                <YAxis 
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Average Price (€)" angle={-90} position="center" dx={-60} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any, name: string, props: any) => {
                                        const roundedValue = Math.round(value / 1000) * 1000;
                                        return [`€${Number(roundedValue).toLocaleString()} (${props.payload.count} properties)`, name];
                                    }}
                                />
                                <Legend 
                                    verticalAlign="bottom"
                                    align="center"
                                    layout="horizontal"
                                    wrapperStyle={{
                                        paddingTop: "20px"
                                    }}
                                />
                                <Bar 
                                    dataKey="avgPrice" 
                                    fill="#8884d8" 
                                    name="Average Price"
                                />
                                <Bar 
                                    dataKey="medianPrice" 
                                    fill="#82ca9d" 
                                    name="Median Price"
                                />
                            </BarChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
};

export default React.memo(PropertyCharts); 