import React, { useState, useEffect } from 'react';
import {
    Box,
    Button,
    Dialog,
    IconButton,
    Paper,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
    CircularProgress,
    Tooltip
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import LocationSearchingIcon from '@mui/icons-material/LocationSearching';
import { MetropolitanArea } from '../types/metropolitan';
import { api } from '../services/api';
import MetropolitanAreaForm from './MetropolitanAreaForm';

const MetropolitanAreaList: React.FC = () => {
    const [areas, setAreas] = useState<MetropolitanArea[]>([]);
    const [openForm, setOpenForm] = useState(false);
    const [selectedArea, setSelectedArea] = useState<MetropolitanArea | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchAreas = async () => {
        try {
            setLoading(true);
            setError(null);
            const data = await api.getMetroAreas();
            setAreas(data || []);
        } catch (error) {
            console.error('Failed to fetch metropolitan areas:', error);
            setError('Failed to load metropolitan areas. Please try again.');
            setAreas([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchAreas();
    }, []);

    const handleEdit = (area: MetropolitanArea) => {
        setSelectedArea(area);
        setOpenForm(true);
    };

    const handleDelete = async (area: MetropolitanArea) => {
        if (window.confirm('Are you sure you want to delete this metropolitan area?')) {
            try {
                await api.deleteMetroArea(area.name);
                await fetchAreas();
            } catch (error) {
                console.error('Failed to delete metropolitan area:', error);
                setError('Failed to delete metropolitan area. Please try again.');
            }
        }
    };

    const handleFormSubmit = async () => {
        await fetchAreas();
        setOpenForm(false);
        setSelectedArea(null);
    };

    const handleGeocodeArea = async (area: MetropolitanArea) => {
        try {
            setLoading(true);
            await api.geocodeMetroArea(area.name);
            await fetchAreas(); // Refresh the list to show updated coordinates
        } catch (error) {
            console.error('Failed to geocode metropolitan area:', error);
            setError('Failed to geocode metropolitan area. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
                <CircularProgress />
            </Box>
        );
    }

    return (
        <Box>
            {error && (
                <Box sx={{ mb: 2 }}>
                    <Typography color="error">{error}</Typography>
                </Box>
            )}
            
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h5">Metropolitan Areas</Typography>
                <Button
                    variant="contained"
                    color="primary"
                    onClick={() => {
                        setSelectedArea(null);
                        setOpenForm(true);
                    }}
                >
                    Add New Area
                </Button>
            </Box>

            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Name</TableCell>
                            <TableCell>Cities</TableCell>
                            <TableCell>Center Coordinates</TableCell>
                            <TableCell>Zoom Level</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {areas.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} align="center">
                                    <Typography color="textSecondary">
                                        No metropolitan areas found. Click "Add New Area" to create one.
                                    </Typography>
                                </TableCell>
                            </TableRow>
                        ) : (
                            areas.map((area) => (
                                <TableRow key={area.name}>
                                    <TableCell>{area.name}</TableCell>
                                    <TableCell>{area.cities.join(', ')}</TableCell>
                                    <TableCell>
                                        {area.center_lat && area.center_lng ? (
                                            `${area.center_lat.toFixed(6)}, ${area.center_lng.toFixed(6)}`
                                        ) : (
                                            <Typography color="textSecondary" variant="body2">
                                                Not geocoded
                                            </Typography>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        {area.zoom_level || (
                                            <Typography color="textSecondary" variant="body2">
                                                Default
                                            </Typography>
                                        )}
                                    </TableCell>
                                    <TableCell align="right">
                                        <Tooltip title="Geocode cities">
                                            <IconButton 
                                                onClick={() => handleGeocodeArea(area)} 
                                                color="primary"
                                                disabled={loading}
                                            >
                                                <LocationSearchingIcon />
                                            </IconButton>
                                        </Tooltip>
                                        <IconButton onClick={() => handleEdit(area)} color="primary">
                                            <EditIcon />
                                        </IconButton>
                                        <IconButton onClick={() => handleDelete(area)} color="error">
                                            <DeleteIcon />
                                        </IconButton>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </TableContainer>

            <Dialog open={openForm} onClose={() => setOpenForm(false)} maxWidth="md" fullWidth>
                <MetropolitanAreaForm
                    area={selectedArea}
                    onSubmit={handleFormSubmit}
                    onCancel={() => setOpenForm(false)}
                />
            </Dialog>
        </Box>
    );
};

export default MetropolitanAreaList; 