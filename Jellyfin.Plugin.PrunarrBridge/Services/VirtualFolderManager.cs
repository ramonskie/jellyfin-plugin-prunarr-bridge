using System;
using System.Linq;
using System.Threading.Tasks;
using Jellyfin.Data.Enums;
using MediaBrowser.Controller.Entities;
using MediaBrowser.Controller.Library;
using MediaBrowser.Model.Configuration;
using Microsoft.Extensions.Logging;

namespace Jellyfin.Plugin.PrunarrBridge.Services;

/// <summary>
/// Service for managing Virtual Folders using Jellyfin's ILibraryManager.
/// </summary>
public class VirtualFolderManager
{
    private readonly ILogger<VirtualFolderManager> _logger;
    private readonly ILibraryManager _libraryManager;

    /// <summary>
    /// Initializes a new instance of the <see cref="VirtualFolderManager"/> class.
    /// </summary>
    /// <param name="logger">The logger.</param>
    /// <param name="libraryManager">The library manager.</param>
    public VirtualFolderManager(ILogger<VirtualFolderManager> logger, ILibraryManager libraryManager)
    {
        _logger = logger;
        _libraryManager = libraryManager;
    }

    /// <summary>
    /// Creates or updates a Virtual Folder.
    /// </summary>
    /// <param name="name">The name of the Virtual Folder.</param>
    /// <param name="collectionType">The collection type (e.g., "movies", "tvshows").</param>
    /// <param name="paths">The paths to add to the Virtual Folder.</param>
    /// <returns>A task representing the asynchronous operation.</returns>
    public async Task CreateOrUpdateVirtualFolderAsync(string name, string collectionType, string[] paths)
    {
        _logger.LogInformation("Creating/updating Virtual Folder: {Name} ({Type})", name, collectionType);

        try
        {
            // Check if Virtual Folder already exists
            var existingFolder = _libraryManager.GetVirtualFolders(false)
                .FirstOrDefault(vf => vf.Name.Equals(name, StringComparison.OrdinalIgnoreCase));

            if (existingFolder != null)
            {
                _logger.LogInformation("Virtual Folder already exists: {Name}", name);
                
                // Update paths if needed
                foreach (var path in paths)
                {
                    if (!existingFolder.Locations.Contains(path))
                    {
                        _logger.LogInformation("Adding path to Virtual Folder: {Path}", path);
                        _libraryManager.AddMediaPath(name, new MediaPathInfo { Path = path });
                    }
                }
            }
            else
            {
                _logger.LogInformation("Creating new Virtual Folder: {Name}", name);
                
                // Create new Virtual Folder
                var libraryOptions = new LibraryOptions
                {
                    EnablePhotos = false,
                    EnableRealtimeMonitor = true,
                    EnableChapterImageExtraction = false,
                    SaveLocalMetadata = false
                };

                // Note: CollectionTypeOptions parameter may need to be null or use string directly
                // depending on Jellyfin API version. For now, using null for mixed content.
                _libraryManager.AddVirtualFolder(name, null, libraryOptions, true);

                // Add paths
                foreach (var path in paths)
                {
                    _logger.LogInformation("Adding path to Virtual Folder: {Path}", path);
                    _libraryManager.AddMediaPath(name, new MediaPathInfo { Path = path });
                }
            }

            _logger.LogInformation("Virtual Folder created/updated successfully");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to create/update Virtual Folder");
            throw;
        }
    }

    /// <summary>
    /// Removes a Virtual Folder.
    /// </summary>
    /// <param name="name">The name of the Virtual Folder to remove.</param>
    /// <returns>A task representing the asynchronous operation.</returns>
    public Task RemoveVirtualFolderAsync(string name)
    {
        _logger.LogInformation("Removing Virtual Folder: {Name}", name);

        try
        {
            _libraryManager.RemoveVirtualFolder(name, true);
            _logger.LogInformation("Virtual Folder removed successfully");
            return Task.CompletedTask;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to remove Virtual Folder");
            throw;
        }
    }

    /// <summary>
    /// Triggers a library scan for the specified Virtual Folder.
    /// </summary>
    /// <param name="name">The name of the Virtual Folder to scan.</param>
    /// <returns>A task representing the asynchronous operation.</returns>
    public async Task ScanLibraryAsync(string name)
    {
        _logger.LogInformation("Scanning library for Virtual Folder: {Name}", name);

        try
        {
            var folder = _libraryManager.GetVirtualFolders(false)
                .FirstOrDefault(vf => vf.Name.Equals(name, StringComparison.OrdinalIgnoreCase));

            if (folder == null)
            {
                throw new ArgumentException($"Virtual Folder not found: {name}");
            }

            // Get the library item for this folder
            var libraryItem = _libraryManager.RootFolder.Children
                .FirstOrDefault(c => c.Name.Equals(name, StringComparison.OrdinalIgnoreCase));

            if (libraryItem != null)
            {
                await _libraryManager.ValidateMediaLibrary(new Progress<double>(), System.Threading.CancellationToken.None);
                _logger.LogInformation("Library scan completed");
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to scan library");
            throw;
        }
    }
}
