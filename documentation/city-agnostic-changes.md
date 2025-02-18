# Making the App City-Agnostic

This document outlines all the places in the codebase where Amsterdam-specific code needs to be modified to make the app city-agnostic.

## Frontend Components

### Map Components
1. **PropertyMap.tsx**
   - Hardcoded Amsterdam center coordinates:
   ```typescript
   const AMSTERDAM_CENTER: LatLngTuple = [52.3676, 4.9041];
   ```

2. **PriceHeatmap.tsx**
   - Hardcoded center coordinates:
   ```typescript
   center={[52.3676, 4.9041]}
   ```

## Backend Components

### SQL Queries
1. **district.go**
   - Hardcoded Amsterdam postal code range:
   ```sql
   AND substr(postal_code, 1, 4) BETWEEN '1000' AND '1999'  -- Ensure valid Amsterdam postal codes
   ```

## Scraping Logic

### Spider Configuration
1. **run_spider.py**
   - Default place parameter:
   ```python
   def run_spider(spider_type, place='amsterdam', max_pages=None, resume=False):
   ```
   ```python
   place = input_data.get('place', 'amsterdam')
   ```

2. **funda_spider.py**
   - Hardcoded city name:
   ```python
   item.city = 'Amsterdam'
   ```
   - Default place parameter:
   ```python
   def __init__(self, place='amsterdam', max_pages=None, *args, **kwargs):
   ```

3. **funda_spider_sold.py**
   - Default place parameter:
   ```python
   def __init__(self, place='amsterdam', max_pages=None, resume=False, *args, **kwargs):
   ```

## App Metadata

### Frontend Metadata
1. **manifest.json**
   ```json
   "name": "FundaMental - Amsterdam Property Analysis"
   ```

2. **index.html**
   ```html
   <meta name="description" content="FundaMental - Amsterdam Property Analysis" />
   ```

3. **App.tsx**
   - Title: "FundaMental - Amsterdam Property Analysis"

## GeoJSON Data

### District Data
1. **district_hulls.geojson**
   - Multiple instances of:
   ```json
   "city": "Amsterdam"
   ```
   - Contains Amsterdam-specific district coordinates

## Required Changes

To make the app city-agnostic, we need to:

1. **Map Configuration**
   - Make map center and zoom level configurable based on selected city
   - Store city coordinates in a configuration file or database

2. **Postal Code Handling**
   - Remove Amsterdam-specific postal code filtering
   - Implement dynamic postal code validation based on city

3. **Scraping Logic**
   - Make city name dynamic in scraping logic
   - Update default parameters to be configurable
   - Implement city-specific scraping rules if needed

4. **App Branding**
   - Update app titles and metadata to be city-agnostic
   - Make city name dynamic in UI elements

5. **GeoJSON Implementation**
   - Make district GeoJSON data dynamic based on selected city
   - Implement system to load city-specific district boundaries
   - Consider implementing dynamic district boundary fetching

## Implementation Strategy

1. **Phase 1: Configuration**
   - Create city configuration system
   - Define city-specific parameters (coordinates, postal codes, etc.)

2. **Phase 2: Backend Updates**
   - Update database schema if needed
   - Modify SQL queries to be city-agnostic
   - Update scraping logic

3. **Phase 3: Frontend Updates**
   - Implement city selection UI
   - Make map components dynamic
   - Update metadata and branding

4. **Phase 4: GeoJSON System**
   - Implement dynamic district boundary loading
   - Create city-specific GeoJSON files
   - Update map visualization components 