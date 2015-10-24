package io.chuie.selfdestruct;

import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.scheduler.BukkitTask;

import java.io.File;

import org.bukkit.command.Command;
import org.bukkit.command.CommandSender;
import org.bukkit.event.EventHandler;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;

public final class SelfDestruct extends JavaPlugin implements Listener {
	private static final long waitTime = 300L;

	private BukkitTask shutdownTask;

	@Override
	public void onEnable() {
		File destroyFlagFile = new File(getServer().getWorldContainer().getAbsoluteFile(), "destroy.txt");
		if (destroyFlagFile.exists()) {
			try {
				destroyFlagFile.delete();
			} catch (Exception e) {
				getLogger().warning("Failed to delete destruction flag file!");
			}
		}

		shutdownTask = getServer().getScheduler().runTaskLater(this, new Runnable() {
			public void run() {
				destroyAndShutdown();
			}
		}, waitTime);
		
		getServer().getPluginManager().registerEvents(this, this);
		getLogger().info("SelfDestruct enabled!");
	}

	@Override
	public void onDisable() {
		getLogger().info("SelfDestruct disabled!");
	}

	@Override
	public boolean onCommand(CommandSender sender, Command cmd, String label, String[] args) {
		if (cmd.getName().equalsIgnoreCase("destruct")) {
			sender.sendMessage("Shutting down and destructing server...");
			if (!destroyAndShutdown()) {
				sender.sendMessage("Could not safely destruct server. Shutdown cancelled!");
			}
			return true;
		}
		return false;
	}

	private boolean destroyAndShutdown() {
		getLogger().info("Shutting down and destructing server...");
		File destroyFlagFile = new File(getServer().getWorldContainer().getAbsoluteFile(), "destroy.txt");
		if (!destroyFlagFile.exists()) {
			try {
				destroyFlagFile.createNewFile();
			} catch (Exception e) {
				getLogger().warning("Failed to create destruction flag file! Server will not shutdown.");
				return false;
			}
		}

		getServer().shutdown();

		return true;
	}

	@EventHandler
	public void onPlayerJoin(PlayerJoinEvent evt) {
		if (shutdownTask != null) {;
			shutdownTask.cancel();
			shutdownTask = null;
		}
	}

	@EventHandler
	public void onPlayerQuit(PlayerQuitEvent evt) {
		if (getServer().getOnlinePlayers().size() == 1) {
			shutdownTask = getServer().getScheduler().runTaskLater(this, new Runnable() {
				public void run() {
					destroyAndShutdown();
				}
			}, waitTime);
		}
	}
}
