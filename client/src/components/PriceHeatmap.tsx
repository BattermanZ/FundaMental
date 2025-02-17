import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, GeoJSON, Tooltip } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import { Property } from '../types/property';
import * as d3 from 'd3';
import { Box, Typography, Paper } from '@mui/material';

interface PriceHeatmapProps {
    properties: Property[];
    metric: 'price' | 'price_per_sqm';
}

interface DistrictData {
    district: string;
    avg_value: number;
    median_value: number;
    count: number;
}

// Legend component
const Legend: React.FC<{ min: number; max: number; metric: string }> = ({ min, max, metric }) => {
    const steps = 5; // Number of color steps
    const values = Array.from({ length: steps }, (_, i) => 
        min + (i * (max - min) / (steps - 1))
    );

    const colorScale = d3.scaleSequential()
        .domain([min, max])
        .interpolator((t) => d3.interpolateRdYlGn(1 - t)); // Reverse the scale to get green-to-red

    const formatValue = (value: number) => {
        if (metric === 'price') {
            return `€${(value/1000).toFixed(0)}k`;
        }
        return `€${value.toFixed(0)}/m²`;
    };

    return (
        <Box
            sx={{
                position: 'absolute',
                bottom: '20px',
                right: '20px',
                backgroundColor: 'white',
                padding: '10px',
                borderRadius: '4px',
                boxShadow: '0 0 10px rgba(0,0,0,0.2)',
                zIndex: 1000,
                minWidth: '200px',
            }}
        >
            <Typography variant="subtitle2" gutterBottom>
                {metric === 'price' ? 'Price Range' : 'Price per m² Range'}
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                {values.reverse().map((value, i) => (
                    <Box key={i} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Box
                            sx={{
                                width: '30px',
                                height: '20px',
                                backgroundColor: colorScale(value),
                                border: '1px solid rgba(0,0,0,0.1)',
                            }}
                        />
                        <Typography variant="caption">
                            {formatValue(value)}
                        </Typography>
                    </Box>
                ))}
            </Box>
        </Box>
    );
};

const PriceHeatmap: React.FC<PriceHeatmapProps> = ({ properties, metric }) => {
    const [districtData, setDistrictData] = useState<Map<string, DistrictData>>(new Map());
    const [geoJsonData, setGeoJsonData] = useState<any>(null);
    const [valueRange, setValueRange] = useState<{ min: number; max: number }>({ min: 0, max: 0 });

    useEffect(() => {
        // Load GeoJSON data for Amsterdam district hulls
        fetch('/district_hulls.geojson')
            .then(response => response.json())
            .then(data => setGeoJsonData(data))
            .catch(error => console.error('Error loading district hulls:', error));
    }, []);

    useEffect(() => {
        // Calculate statistics for each district
        const districtGroups = d3.group(
            properties.filter(p => p.price && (metric === 'price' || p.living_area)),
            d => d.postal_code.substring(0, 4)
        );

        const newDistrictData = new Map<string, DistrictData>();
        const values: number[] = [];
        
        districtGroups.forEach((group, district) => {
            const districtValues = group.map(p => metric === 'price' ? p.price : (p.price / (p.living_area || 1)));
            const avgValue = d3.mean(districtValues) || 0;
            values.push(avgValue);
            
            newDistrictData.set(district, {
                district,
                avg_value: avgValue,
                median_value: d3.median(districtValues) || 0,
                count: group.length
            });
        });

        setDistrictData(newDistrictData);
        setValueRange({
            min: d3.min(values) || 0,
            max: d3.max(values) || 0
        });
    }, [properties, metric]);

    if (!geoJsonData) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="400px">
                <Typography>Loading map data...</Typography>
            </Box>
        );
    }

    const getColor = (district: string) => {
        const data = districtData.get(district);
        if (!data) return '#cccccc';

        // Create a color scale from green to red using RdYlGn
        const colorScale = d3.scaleSequential()
            .domain([valueRange.min, valueRange.max])
            .interpolator((t) => d3.interpolateRdYlGn(1 - t)); // Reverse the scale to get green-to-red

        return colorScale(data.avg_value);
    };

    const style = (feature: any) => {
        return {
            fillColor: getColor(feature.properties.district),
            weight: 2,
            opacity: 1,
            color: 'white',
            fillOpacity: 0.7
        };
    };

    const formatValue = (value: number) => {
        if (metric === 'price') {
            return `€${(value/1000).toFixed(0)}k`;
        }
        return `€${value.toFixed(0)}/m²`;
    };

    const onEachFeature = (feature: any, layer: any) => {
        const data = districtData.get(feature.properties.district);
        if (data) {
            layer.bindTooltip(`
                <strong>District: ${data.district}</strong><br/>
                Average ${metric === 'price' ? 'Price' : 'Price/m²'}: ${formatValue(data.avg_value)}<br/>
                Median ${metric === 'price' ? 'Price' : 'Price/m²'}: ${formatValue(data.median_value)}<br/>
                Number of properties: ${data.count}
            `);
        }
    };

    return (
        <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
                {metric === 'price' ? 'Price' : 'Price per Square Meter'} by District
            </Typography>
            <Box sx={{ height: 500, width: '100%', position: 'relative' }}>
                <MapContainer
                    center={[52.3676, 4.9041]}
                    zoom={13}
                    style={{ height: '100%', width: '100%' }}
                >
                    <TileLayer
                        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                    />
                    <GeoJSON
                        data={geoJsonData}
                        style={style}
                        onEachFeature={onEachFeature}
                    />
                </MapContainer>
                <Legend min={valueRange.min} max={valueRange.max} metric={metric} />
            </Box>
        </Paper>
    );
};

export default PriceHeatmap; 