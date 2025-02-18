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
    Typography
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { MetropolitanArea } from '../types/metropolitan';
import { api } from '../services/api';
import MetropolitanAreaForm from './MetropolitanAreaForm';

const MetropolitanAreaList: React.FC = () => {
    const [areas, setAreas] = useState<MetropolitanArea[]>([]);
    const [openForm, setOpenForm] = useState(false);
    const [selectedArea, setSelectedArea] = useState<MetropolitanArea | null>(null);
    const [loading, setLoading] = useState(true);

    const fetchAreas = async () => {
        try {
            const data = await api.getMetropolitanAreas();
            setAreas(data);
        } catch (error) {
            console.error('Failed to fetch metropolitan areas:', error);
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
                await api.deleteMetropolitanArea(area.name);
                await fetchAreas();
            } catch (error) {
                console.error('Failed to delete metropolitan area:', error);
            }
        }
    };

    const handleFormSubmit = async () => {
        await fetchAreas();
        setOpenForm(false);
        setSelectedArea(null);
    };

    if (loading) {
        return (
            <Box sx={{ p: 2 }}>
                <Typography>Loading metropolitan areas...</Typography>
            </Box>
        );
    }

    return (
        <Box>
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
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {areas.map((area) => (
                            <TableRow key={area.name}>
                                <TableCell>{area.name}</TableCell>
                                <TableCell>{area.cities.join(', ')}</TableCell>
                                <TableCell align="right">
                                    <IconButton onClick={() => handleEdit(area)} color="primary">
                                        <EditIcon />
                                    </IconButton>
                                    <IconButton onClick={() => handleDelete(area)} color="error">
                                        <DeleteIcon />
                                    </IconButton>
                                </TableCell>
                            </TableRow>
                        ))}
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