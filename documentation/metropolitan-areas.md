# Metropolitan Areas Implementation

## Overview
This document outlines a simple, user-defined approach for handling metropolitan areas in the property tracking system. The focus is on allowing users to define which cities belong to a metropolitan area (e.g., Amsterdam metropolitan area including Amstelveen and Diemen) through the frontend interface.

## Core Concept
Instead of managing complex postal code ranges or geographic boundaries, we'll use a simple city-based grouping system where:
- Users define metropolitan areas through the frontend
- Cities can be added to or removed from metropolitan areas dynamically
- Queries automatically include all cities in a metropolitan area when requested

## Data Structure

### Simple Metropolitan Area Definition
```go
type MetropolitanArea struct {
    Name     string   // e.g., "Amsterdam"
    Cities   []string // e.g., ["Amsterdam", "Amstelveen", "Diemen"]
}
```

## Implementation Strategy

1. **Frontend Configuration**
   - Add a metropolitan area management interface
   - Allow users to:
     - Create metropolitan areas
     - Add/remove cities from metropolitan areas
     - View current metropolitan area configurations

2. **Query Handling**
   - When querying for a metropolitan area:
     - Include all properties from constituent cities
     - Maintain original city names in the database
     - Aggregate statistics across all included cities

3. **Spider Integration**
   - When scraping a metropolitan area:
     - Run spider for each constituent city
     - Maintain original city names in the data
     - No need for postal code management

## Benefits
1. Simple to implement and maintain
2. Flexible - cities can be added or removed easily
3. No database schema changes required
4. Clear and understandable for users
5. No need for complex postal code management
6. Easy to extend to other metropolitan areas

## Example Usage

### Frontend Interface
```typescript
interface MetropolitanAreaConfig {
    name: string;
    cities: string[];
}

// Example configuration
const amsterdamMetro: MetropolitanAreaConfig = {
    name: "Amsterdam Metropolitan Area",
    cities: ["Amsterdam", "Amstelveen", "Diemen"]
};
```

### Query Handling
- When user selects "Amsterdam Metropolitan Area":
  - Backend automatically includes properties from all constituent cities
  - Statistics are aggregated across all included cities
  - Original city names are preserved in the data

## Integration Points

### API Changes
- New endpoints for:
  - Managing metropolitan area definitions
  - Querying metropolitan area data
  - Getting aggregated statistics

### Spider Integration
- Spider manager handles multiple cities sequentially
- No changes needed to spider logic
- Cities maintain their original identities

## Limitations
1. Relies on user configuration
2. Manual updates required when metropolitan area composition changes
3. No automatic geographic validation

## Next Steps
1. Design metropolitan area management UI
2. Implement metropolitan area configuration endpoints
3. Update query handling to support metropolitan areas
4. Add metropolitan area support to statistics calculations 