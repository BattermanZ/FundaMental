# Metropolitan Areas Implementation

## Overview
This document outlines the implementation of metropolitan areas in the system, including coordinate management and map integration.

## Architecture

### 1. Data Models
Located in `server/internal/models/metropolitan.go`:
- `MetropolitanArea`: Core model with coordinates and zoom level
- `MetropolitanCity`: Individual city data with coordinates
- `MetropolitanConfig`: Configuration file structure

### 2. Configuration Management
Located in `server/config/metropolitan.go`:
- JSON-based configuration for initial setup
- Thread-safe operations with mutex locks
- Functions for loading/saving configurations

### 3. API Layer
Located in `server/internal/api/metropolitan.go`:
- RESTful endpoints for CRUD operations
- Geocoding integration
- Automatic center calculation

## API Endpoints

### Metropolitan Area Management
- `GET /api/metropolitan`: List all areas
- `POST /api/metropolitan`: Create new area
- `GET /api/metropolitan/:name`: Get specific area
- `PUT /api/metropolitan/:name`: Update area
- `DELETE /api/metropolitan/:name`: Delete area
- `POST /api/metropolitan/:name/geocode`: Geocode cities in area

## Geocoding Implementation

### Process Flow
1. User creates/updates metropolitan area with cities
2. Geocoding can be triggered manually via UI button
3. Each city is geocoded using Nominatim API
4. Coordinates are validated against Netherlands bounds
5. Results are cached to minimize API calls
6. Area center is automatically calculated

### Geocoding Service
- Rate limited to 1 request/second
- Persistent cache with file storage
- Validates coordinates within Netherlands bounds
- Handles errors gracefully with logging

### Coordinate Management
- Cities stored with individual coordinates
- Metropolitan center calculated as geometric mean
- Zoom level can be customized per area
- Coordinates updated asynchronously

## Frontend Integration

### Components
- `MetropolitanAreaList`: Main management interface
- `MetropolitanAreaForm`: Creation/editing form
- Interactive geocoding with status feedback

### Features
- Display of coordinates and zoom levels
- Manual geocoding trigger button
- Error handling and loading states
- Automatic list refresh after updates

## Error Handling

### Validation
- City name validation
- Coordinate bounds checking
- Required field validation
- Duplicate prevention

### Error Responses
- Clear error messages
- Loading state indication
- Geocoding failure handling
- Network error recovery

## Future Improvements

### Planned Features
- Batch geocoding operations
- Population-weighted center calculation
- Automatic zoom level optimization
- Advanced caching strategies

### Performance Optimizations
- Parallel geocoding with rate limiting
- Precomputed statistics caching
- Efficient coordinate updates
- Bulk operation support

## Configuration Migration
The system supports migration from static configuration to database storage:
1. Initial setup from JSON configuration
2. Database storage for dynamic updates
3. Coordinate persistence
4. Automatic center recalculation
