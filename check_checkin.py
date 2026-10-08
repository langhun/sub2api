
def main():
    """主函数"""
    print("签到异常行为检测工具")
    print(f"连接数据库: {DB_HOST}:{DB_PORT}/{DB_NAME}")

    try:
        conn = get_db_connection()
        print("数据库连接成功")

        # 1. 检查指定用户
        check_user_checkin(conn, 'jichun29@163.com')

        # 2. 查找同一时间签到模式
        find_same_time_patterns(conn)

        # 3. 查找高连续签到用户
        find_high_streak_users(conn)

        # 4. 检查IP模式（如果有）
        check_ip_patterns(conn, 'jichun29@163.com')

        conn.close()
        print(f"\n{'='*70}")
        print("检查完成")
        print(f"{'='*70}")

    except pymysql.Error as e:
        print(f"数据库错误: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"错误: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == '__main__':
    main()
