package tech.coffers;

import org.slf4j.Marker;
import org.slf4j.MarkerFactory;

/**
 * Standard Markers for NanoLog control.
 * Usage:
 * log.info(NanoLogMarkers.IGNORE, "This log will be skipped by NanoLog");
 */
public class NanoLogMarkers {

    /**
     * Marker to indicate that this log event should NOT be sent to NanoLog server.
     * It will still appear in other appenders (like Console/File) if configured.
     */
    public static final Marker IGNORE = MarkerFactory.getMarker("NANOLOG_IGNORE");

}
