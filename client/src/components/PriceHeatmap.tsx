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

const PriceHeatmap: React.FC<PriceHeatmapProps> = ({ properties, metric }) => {
    const [districtData, setDistrictData] = useState<Map<string, DistrictData>>(new Map());
    const [geoJsonData, setGeoJsonData] = useState<any>(null);

    useEffect(() => {
        // Load GeoJSON data for Amsterdam district hulls
        fetch('/amsterdam_district_hulls.geojson')
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
        
        districtGroups.forEach((group, district) => {
            const values = group.map(p => metric === 'price' ? p.price : (p.price / (p.living_area || 1)));
            newDistrictData.set(district, {
                district,
                avg_value: d3.mean(values) || 0,
                median_value: d3.median(values) || 0,
                count: group.length
            });
        });

        setDistrictData(newDistrictData);
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

        // Create a color scale from green to red
        const colorScale = d3.scaleSequential()
            .domain([0, d3.max(Array.from(districtData.values()), d => d.avg_value) || 0])
            .interpolator(d3.interpolateRdYlBu);

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
            <Box sx={{ height: 500, width: '100%' }}>
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
            </Box>
        </Paper>
    );
};

export default PriceHeatmap; 