let cachedGrid = null;
let cachedPredictionData = null;
let cachedYearOffset = 0;

function getElevationColor(elevation, minElev, maxElev) {
    const range = maxElev - minElev || 1;
    const normalized = (elevation - minElev) / range;

    let r, g, b;

    if (normalized < 0.25) {
        const t = normalized / 0.25;
        r = 0;
        g = Math.floor(100 + t * 50);
        b = Math.floor(200 + t * 55);
    } else if (normalized < 0.5) {
        const t = (normalized - 0.25) / 0.25;
        r = Math.floor(t * 100);
        g = Math.floor(150 + t * 105);
        b = Math.floor(255 - t * 100);
    } else if (normalized < 0.75) {
        const t = (normalized - 0.5) / 0.25;
        r = Math.floor(100 + t * 155);
        g = Math.floor(255 - t * 55);
        b = Math.floor(155 - t * 100);
    } else {
        const t = (normalized - 0.75) / 0.25;
        r = Math.floor(255 - t * 55);
        g = Math.floor(200 - t * 100);
        b = Math.floor(55 + t * 50);
    }

    return [r, g, b];
}

function getAdjustedElevation(baseElevation, predictionData, yearOffset) {
    if (!predictionData || predictionData.length === 0) {
        return baseElevation;
    }

    const monthIndex = Math.min(
        Math.floor(yearOffset),
        predictionData.length - 1
    );
    const prediction = predictionData[monthIndex];

    if (prediction && prediction.bed_elevation_change !== undefined) {
        return baseElevation + prediction.bed_elevation_change;
    }

    return baseElevation;
}

function computeElevationMap(elevations, predictionData, yearOffset, width, height) {
    const rows = elevations.length;
    const cols = elevations[0].length;
    const cellWidth = width / cols;
    const cellHeight = height / rows;

    let minElev = Infinity;
    let maxElev = -Infinity;

    const adjusted = new Array(rows);
    for (let j = 0; j < rows; j++) {
        adjusted[j] = new Float64Array(cols);
        for (let i = 0; i < cols; i++) {
            const elev = getAdjustedElevation(elevations[j][i], predictionData, yearOffset);
            adjusted[j][i] = elev;
            if (elev < minElev) minElev = elev;
            if (elev > maxElev) maxElev = elev;
        }
    }

    const imageData = new Uint8ClampedArray(width * height * 4);

    for (let j = 0; j < rows; j++) {
        for (let i = 0; i < cols; i++) {
            const elevation = adjusted[j][i];
            const [r, g, b] = getElevationColor(elevation, minElev, maxElev);

            const startX = Math.floor(i * cellWidth);
            const endX = Math.floor((i + 1) * cellWidth) + 1;
            const startY = Math.floor(j * cellHeight);
            const endY = Math.floor((j + 1) * cellHeight) + 1;

            for (let py = startY; py < endY && py < height; py++) {
                for (let px = startX; px < endX && px < width; px++) {
                    const idx = (py * width + px) * 4;
                    imageData[idx] = r;
                    imageData[idx + 1] = g;
                    imageData[idx + 2] = b;
                    imageData[idx + 3] = 255;
                }
            }
        }
    }

    return {
        imageData: imageData.buffer,
        width: width,
        height: height,
        minElev: minElev,
        maxElev: maxElev,
        adjusted: adjusted,
        cellWidth: cellWidth,
        cellHeight: cellHeight
    };
}

function computeContourLines(elevations, predictionData, yearOffset, minElev, maxElev, cellWidth, cellHeight, canvasWidth, canvasHeight) {
    const contourInterval = 0.5;
    const startElev = Math.ceil(minElev / contourInterval) * contourInterval;

    const segments = [];

    for (let elev = startElev; elev <= maxElev; elev += contourInterval) {
        for (let j = 0; j < elevations.length - 1; j++) {
            for (let i = 0; i < elevations[0].length - 1; i++) {
                const e00 = getAdjustedElevation(elevations[j][i], predictionData, yearOffset);
                const e10 = getAdjustedElevation(elevations[j][i + 1], predictionData, yearOffset);
                const e01 = getAdjustedElevation(elevations[j + 1][i], predictionData, yearOffset);
                const e11 = getAdjustedElevation(elevations[j + 1][i + 1], predictionData, yearOffset);

                const crossings = [];

                if ((e00 - elev) * (e10 - elev) < 0) {
                    const t = (elev - e00) / (e10 - e00);
                    crossings.push({ x: (i + t) * cellWidth, y: j * cellHeight });
                }
                if ((e10 - elev) * (e11 - elev) < 0) {
                    const t = (elev - e10) / (e11 - e10);
                    crossings.push({ x: (i + 1) * cellWidth, y: (j + t) * cellHeight });
                }
                if ((e01 - elev) * (e11 - elev) < 0) {
                    const t = (elev - e01) / (e11 - e01);
                    crossings.push({ x: (i + t) * cellWidth, y: (j + 1) * cellHeight });
                }
                if ((e00 - elev) * (e01 - elev) < 0) {
                    const t = (elev - e00) / (e01 - e00);
                    crossings.push({ x: i * cellWidth, y: (j + t) * cellHeight });
                }

                if (crossings.length >= 2) {
                    segments.push({
                        x1: crossings[0].x,
                        y1: crossings[0].y,
                        x2: crossings[1].x,
                        y2: crossings[1].y
                    });
                }
            }
        }
    }

    return segments;
}

self.onmessage = function(e) {
    const { type, payload } = e.data;

    switch (type) {
        case 'setGrid': {
            cachedGrid = payload.grid;
            cachedPredictionData = payload.predictionData || null;
            cachedYearOffset = payload.yearOffset || 0;
            break;
        }

        case 'setPredictionData': {
            cachedPredictionData = payload.predictionData;
            break;
        }

        case 'setYearOffset': {
            cachedYearOffset = payload.yearOffset;

            if (cachedGrid) {
                const result = computeElevationMap(
                    cachedGrid.elevations,
                    cachedPredictionData,
                    cachedYearOffset,
                    payload.width,
                    payload.height
                );

                const contours = computeContourLines(
                    cachedGrid.elevations,
                    cachedPredictionData,
                    cachedYearOffset,
                    result.minElev,
                    result.maxElev,
                    result.cellWidth,
                    result.cellHeight,
                    payload.width,
                    payload.height
                );

                self.postMessage({
                    type: 'renderComplete',
                    payload: {
                        imageData: result.imageData,
                        width: result.width,
                        height: result.height,
                        minElev: result.minElev,
                        maxElev: result.maxElev,
                        contours: contours
                    }
                }, [result.imageData]);
            }
            break;
        }

        case 'fullRender': {
            if (!cachedGrid) {
                self.postMessage({ type: 'renderComplete', payload: null });
                return;
            }

            const result = computeElevationMap(
                cachedGrid.elevations,
                cachedPredictionData,
                cachedYearOffset,
                payload.width,
                payload.height
            );

            const contours = computeContourLines(
                cachedGrid.elevations,
                cachedPredictionData,
                cachedYearOffset,
                result.minElev,
                result.maxElev,
                result.cellWidth,
                result.cellHeight,
                payload.width,
                payload.height
            );

            self.postMessage({
                type: 'renderComplete',
                payload: {
                    imageData: result.imageData,
                    width: result.width,
                    height: result.height,
                    minElev: result.minElev,
                    maxElev: result.maxElev,
                    contours: contours
                }
            }, [result.imageData]);
            break;
        }
    }
};
