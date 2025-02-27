import React, { useEffect, useState, useMemo, useCallback, useDeferredValue } from 'react';
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
import FilterPanel, { FilterOptions } from './FilterPanel';
import PropertyChartData from './PropertyChartData';

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

    // Use deferred value for expensive computations
    const deferredFilters = useDeferredValue(filters);

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

    // Memoize filtered properties using deferred filters
    const filteredPropertiesMemo = useMemo(() => {
        if (!properties || properties.length === 0) return [];
        return properties.filter(property => {
            // Date filter - check listing_date/scraped_at for active and selling_date for sold
            if (deferredFilters.startDate) {
                const effectiveDate = property.status === 'active' 
                    ? (property.listing_date ? dayjs(property.listing_date) : dayjs(property.scraped_at))
                    : (property.selling_date ? dayjs(property.selling_date) : null);
                
                if (effectiveDate && effectiveDate.isBefore(deferredFilters.startDate)) {
                    return false;
                }
            }
            
            if (deferredFilters.endDate) {
                const effectiveDate = property.status === 'active' 
                    ? (property.listing_date ? dayjs(property.listing_date) : dayjs(property.scraped_at))
                    : (property.selling_date ? dayjs(property.selling_date) : null);
                
                if (effectiveDate && effectiveDate.isAfter(deferredFilters.endDate)) {
                    return false;
                }
            }
            
            if (deferredFilters.propertyType !== 'all' && property.property_type !== deferredFilters.propertyType) return false;
            if (deferredFilters.status !== 'all' && property.status !== deferredFilters.status) return false;
            if (property.num_rooms && (
                property.num_rooms < deferredFilters.numRooms[0] ||
                property.num_rooms > deferredFilters.numRooms[1]
            )) return false;
            if (property.price && (
                property.price < deferredFilters.priceRange[0] ||
                property.price > deferredFilters.priceRange[1]
            )) return false;
            if (property.living_area && (
                property.living_area < deferredFilters.sizeRange[0] ||
                property.living_area > deferredFilters.sizeRange[1]
            )) return false;
            
            return true;
        });
    }, [properties, deferredFilters]);

    // Memoize chart data computations
    const chartData = useMemo(() => {
        // Scatter plot data
        const scatter = filteredPropertiesMemo
            .filter(p => p.living_area && p.price)
            .map(p => ({
                living_area: p.living_area,
                price: p.price,
                postal_code: p.postal_code.substring(0, 4),
                num_rooms: p.num_rooms || 1,
                price_per_sqm: p.price / p.living_area
            }));

        // Price by postal code data
        const postalGroups = d3.group(
            filteredPropertiesMemo.filter(p => p.price),
            d => d.postal_code.substring(0, 4)
        );

        const priceByPostal = Array.from(postalGroups, ([postal_code, group]) => ({
            postal_code,
            avg_price: d3.mean(group, d => d.price) || 0,
            median_price: d3.median(group, d => d.price) || 0,
            count: group.length
        })).sort((a, b) => b.avg_price - a.avg_price);

        // Time series data
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

        const soldMonthlyGroups = d3.group(soldProperties, d => 
            dayjs(d.selling_date).format('YYYY-MM')
        );

        const activeMonthlyGroups = d3.group(activeProperties, d => 
            dayjs(d.effectiveDate).format('YYYY-MM')
        );

        const uniqueMonths = new Set([
            ...Array.from(soldMonthlyGroups.keys()),
            ...Array.from(activeMonthlyGroups.keys())
        ]);

        const timeSeries = Array.from(uniqueMonths).sort().map(month => {
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

        // Price per Square Meter Analysis
        const pricePerSqm = Array.from(postalGroups, ([postal_code, group]) => ({
            postal_code,
            avg_price_per_sqm: d3.mean(group.filter(p => p.living_area), d => d.price / d.living_area) || 0,
            median_price_per_sqm: d3.median(group.filter(p => p.living_area), d => d.price / d.living_area) || 0,
            count: group.length
        })).sort((a, b) => b.avg_price_per_sqm - a.avg_price_per_sqm);

        // Rooms Impact Analysis
        const roomGroups = d3.group(
            filteredPropertiesMemo.filter(p => p.num_rooms && p.price),
            d => d.num_rooms
        );

        const roomsImpact = Array.from(roomGroups, ([rooms, group]) => ({
            rooms: Number(rooms),
            count: group.length,
            avg_price: d3.mean(group, d => d.price) || 0,
            median_price: d3.median(group, d => d.price) || 0
        })).sort((a, b) => a.rooms - b.rooms);

        // Room Price Premium Analysis
        const roomPricePremium = roomsImpact.map((item, index) => ({
            rooms: item.rooms,
            count: item.count,
            median_price: item.median_price,
            pricePremium: index > 0 ? item.avg_price - roomsImpact[index - 1].avg_price : 0,
            percentageIncrease: index > 0 ? ((item.avg_price - roomsImpact[index - 1].avg_price) / roomsImpact[index - 1].avg_price) * 100 : 0
        })).filter(d => d.rooms <= 10);

        // Calculate regression line
        const xValues = scatter.map(d => d.living_area);
        const yValues = scatter.map(d => d.price);
        
        const xMean = d3.mean(xValues) || 0;
        const yMean = d3.mean(yValues) || 0;
        
        const ssXX = d3.sum(xValues, x => Math.pow(x - xMean, 2));
        const ssXY = d3.sum(scatter, d => (d.living_area - xMean) * (d.price - yMean));
        
        const slope = ssXY / ssXX;
        const intercept = yMean - slope * xMean;
        
        const minX = Math.min(...xValues);
        const maxX = Math.max(...xValues);
        
        const regressionLine = [
            { x: minX, y: slope * minX + intercept },
            { x: maxX, y: slope * maxX + intercept }
        ];

        return {
            scatterData: scatter,
            priceByPostalCodeData: priceByPostal,
            timeSeriesData: timeSeries,
            regressionLine,
            priceByRoomsData: roomsImpact,
            priceByRoomsPremiumData: roomPricePremium,
            pricePerSqmData: pricePerSqm
        };
    }, [filteredPropertiesMemo]);

    // Handlers for filter panel
    const handleFilterChange = useCallback((newFilters: FilterOptions) => {
        setPendingFilters(newFilters);
    }, []);

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
            <FilterPanel
                pendingFilters={pendingFilters}
                ranges={ranges}
                onFilterChange={handleFilterChange}
                onApplyFilters={applyFilters}
                onResetFilters={resetFilters}
            />
            <Grid container spacing={3}>
                {/* Price Heatmap */}
                <Grid item xs={12}>
                    <PriceHeatmap properties={filteredPropertiesMemo} />
                </Grid>

                {/* Charts */}
                <PropertyChartData {...chartData} />
            </Grid>
        </Box>
    );
};

export default PropertyCharts; 