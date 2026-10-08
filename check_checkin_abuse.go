
func findHighStreakUsers(ctx context.Context, db *sql.DB) {
	query := `
		SELECT u.email, u.id, MAX(c.streak_days) as max_streak,
		       COUNT(*) as total_checkins
		FROM checkins c JOIN users u ON c.user_id = u.id
		GROUP BY u.id
		HAVING max_streak >= 30 OR total_checkins >= 60
		ORDER BY max_streak DESC LIMIT 20
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("Email\t\t\t\tMaxStreak\tTotal")
	for rows.Next() {
		var email string
		var userID int64
		var maxStreak, totalCheckins int
		err := rows.Scan(&email, &userID, &maxStreak, &totalCheckins)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("%s\t%d\t\t%d\n", email, maxStreak, totalCheckins)
	}
}
