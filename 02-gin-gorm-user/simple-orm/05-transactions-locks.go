/*
 * File: 05-transactions-locks.go
 * Date: 2025/10/7 19:29
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_orm

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
	事务是数据库中一组要么全部成功，要么全部失败的操作，保证数据一致性
	简称 ACID 四特性
		A (Atomicity) 原子性     全部成功或全部回滚
		C (Consistency) 一致性   状态从一个合法状态转到另一个合法状态
		I (Isolation) 隔离性     多事务并发互不干扰
		D (Durability) 持久性    提交后数据永久保存
	事务的隔离级别（Isolation Level）
		Read Uncommitted    可以读到别人未提交的    脏读
		Read Committed      只能读到别人已提交的    防止脏读
		Repeatable Read     多次读取结果一致        防止脏读+不可重复读
		Serializable        强制串行执行           防止所有并发问题（最安全但最慢）
		MySQL InnoDB 默认：Repeatable Read; Postgres SQL 默认：Read Committed
	行级锁（SELECT FOR UPDATE）
		在并发转账 / 库存扣减中，可能多个事务同时修改同一行数据
		例子：两个事务几乎同时执行 SELECT balance FROM account WHERE id=1;
		都读到余额=100，然后各自减 100 → 写回 balance=0；结果：两人都成功，余额错乱（超额消费）
		解决：加行锁 SELECT * FROM account WHERE id=1 FOR UPDATE;
			这会在事务内锁住该行，其他事务必须等它 COMMIT/ROLLBACK 才能访问
			同一行只能被一个事务锁定修改，保证余额不会被重复扣
	priority: 是 GORM 为联合索引（uniqueIndex / index）提供的列顺序控制参数
		它的作用是指定当前字段在联合索引中的位置顺序, 它只在同一个索引名出现多次时才起作用
		在 MySQL 中，一个联合索引的顺序是有意义的
		如：
			CREATE UNIQUE INDEX uk_user_biz ON wallet_log(user_id, biz_key);
			CREATE UNIQUE INDEX uk_user_biz ON wallet_log(biz_key, user_id);
		上面2个是不同索引结构
			第一种：先按 user_id 排，再按 biz_key
			第二种：先按 biz_key 排，再按 user_id
		同一索引名出现多次时，必须显式指定 priority
		priority:1 联合索引中的第 1 列, 2就是联合索引中的第 2 列...以此类推
	联合唯一键：它的核心作用并不在查询性能，而在于 「数据唯一性约束 + 幂等保护 + 局部查询优化」
		UNIQUE KEY `uk_user_biz` (`user_id`,`biz_key`)
		表示数据库层面定义了一个 联合唯一约束（Composite Unique Constraint）它确保在这张表中，任意两行的 (user_id, biz_key) 组合值 不能重复
		作用一：防重复写入（幂等性保障）
			在并发/分布式系统中，同一个业务请求可能被重放或多次到达，数据库自动拒绝重复写入，从根源防止重复问题
		作用二：查询性能
			UNIQUE KEY 本质上就是一个带唯一性约束的 B+Tree 索引
			当查询类型：WHERE user_id=? AND biz_key=?  精确匹配命中
			当查询类型：WHERE user_id=?  左前缀匹配（可走范围扫描）
			当查询类型：WHERE biz_key=?  不能单独利用（非最左前缀）
		作用三：并发控制（防止写冲突）
			唯一键在数据库层是通过 行级锁 + 唯一约束检查 实现的
			在并发写时，MySQL 会 插入前先查唯一索引、若冲突，则抛异常、若无冲突，则插入并持有锁
			这天然防止了并发条件竞争（race condition）下的重复入库，所以它也是一种轻量级分布式锁机制（适合单实例 MySQL）
	普通索引（INDEX）
		如 INDEX idx_user_time (user_id,created_at) 性能优化：让查询更快，没有唯一性要求
		插入直接追加（略快），但是UNIQUE KEY 插入前需查冲突（稍慢）
		适用场景：按用户查看最近流水分页，按时间统计所有用户的新增流水
*/

// 事务的两种使用方式（GORM）

// Reason 枚举常量
const (
	ReasonTransfer = "transfer" // 转账
	ReasonDeposit  = "deposit"  // 充值/入金
	ReasonWithdraw = "withdraw" // 提现/出金
	ReasonRefund   = "refund"   // 退款
	ReasonAdjust   = "adjust"   // 调账
)

// WalletLog 钱包流水表（强幂等、可审计、可对账）
type WalletLog struct {
	// uk_user_biz: 自定义的唯一索引名称，只要同一张表内不重复即可
	// uniqueIndex:uk_user_biz: 会生成 UNIQUE KEY uk_user_biz (user_id, biz_key)。如果不写名字会生成一个自动命名的唯一索引
	ID        uint64          `gorm:"primaryKey;autoIncrement;type:bigint unsigned;comment:主键ID"`
	UserID    uint64          `gorm:"type:bigint unsigned;not null;uniqueIndex:uk_user_biz,priority:1;index:idx_user_time,priority:1;comment:关联用户ID"`
	BizKey    string          `gorm:"type:varchar(128);not null;uniqueIndex:uk_user_biz,priority:2;comment:幂等唯一键(与UserID联合唯一)"`
	RefID     string          `gorm:"type:varchar(64);index;comment:业务关联单号"`
	Direction int8            `gorm:"type:tinyint;not null;comment:方向(1入账,-1出账)"`
	Amount    decimal.Decimal `gorm:"type:decimal(18,2) unsigned;not null;comment:本次变动金额"`
	Before    decimal.Decimal `gorm:"type:decimal(18,2) unsigned;not null;comment:变动前余额"`
	After     decimal.Decimal `gorm:"type:decimal(18,2) unsigned;not null;comment:变动后余额"`
	Reason    string          `gorm:"type:varchar(64);not null;index;comment:业务原因(枚举)"`
	// datatypes 是 GORM 提供的常用数据库类型扩展包，常用来存 JSON 字段、时间
	// datatypes.JSON：底层是 []byte，适合你自己管理序列化
	// datatypes.JSONMap：底层是 map[string]interface{}，更易用，GORM 会自动序列化/反序列化
	Meta      datatypes.JSONMap `gorm:"type:json;comment:扩展元数据(JSON)"`
	CreatedAt time.Time         `gorm:"autoCreateTime;index:idx_user_time,priority:2;comment:创建时间(流水发生时间)"`
}

func (WalletLog) TableName() string { return "uu_wallet_log" }

func TransferTx(db *gorm.DB, fromUID, toUID, bizKey string, amount decimal.Decimal) error {
	return db.Transaction(func(tx *gorm.DB) error {
		refId := uuid.New().String()
		// 查出转出与转入用户，并加行锁（FOR UPDATE）
		var from, to User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", fromUID).First(&from).Error; err != nil {
			return fmt.Errorf("转出方不存在: %w", err)
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", toUID).First(&to).Error; err != nil {
			return fmt.Errorf("转入方不存在: %w", err)
		}
		// 余额查询
		if from.Balance.LessThan(amount) {
			return fmt.Errorf("余额不足")
		}
		// 业务幂等判断：尝试插入流水（若存在则返回成功）
		outLog := WalletLog{
			UserID:    from.ID,
			BizKey:    bizKey,
			RefID:     refId,
			Direction: -1,
			Amount:    amount,
			Before:    from.Balance,
			After:     from.Balance.Sub(amount),
			Reason:    ReasonTransfer,
			Meta:      datatypes.JSONMap{"role": "out", "from": fromUID, "to": toUID},
		}
		if err := tx.Create(&outLog).Error; err != nil {
			if strings.Contains(err.Error(), "Duplicate") {
				// 重复转账请求 → 幂等处理：直接返回 nil 表示事务已处理
				return nil
			}
			return err
		}
		// 更新余额
		from.Balance = from.Balance.Sub(amount)
		to.Balance = to.Balance.Add(amount)
		if err := tx.Model(&User{}).
			Where("id = ?", from.ID).
			Update("balance", from.Balance).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).
			Where("id = ?", to.ID).
			Update("balance", to.Balance).Error; err != nil {
			return err
		}
		// 写入入账流水
		inLog := WalletLog{
			UserID: to.ID,
			// BizKey:    bizKey + ":IN",
			BizKey:    bizKey,
			RefID:     refId,
			Direction: 1,
			Amount:    amount,
			Before:    to.Balance.Sub(amount),
			After:     to.Balance,
			Reason:    ReasonTransfer,
			Meta:      datatypes.JSONMap{"role": "in", "from": fromUID, "to": toUID},
		}
		if err := tx.Create(&inLog).Error; err != nil {
			return err
		}
		// 自动提交
		return nil
	})
}

func TransferManual(db *gorm.DB, fromUID, toUID, bizKey string, amount decimal.Decimal) (err error) {
	tx := db.Begin() // 开启事务
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			// 继续抛出，或转为 error
			panic(p)
		} else if err != nil {
			_ = tx.Rollback() // 只有当 err!=nil 时才回滚
		}
	}()

	// 锁定两行（FOR UPDATE）
	var from, to User
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uid = ?", fromUID).First(&from).Error; err != nil {
		return fmt.Errorf("转出方不存在: %w", err)
	}
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uid = ?", toUID).First(&to).Error; err != nil {
		return fmt.Errorf("转入方不存在: %w", err)
	}

	// 余额校验
	if from.Balance.LessThan(amount) {
		return fmt.Errorf("余额不足")
	}

	// 记录 before 值
	fromBefore := from.Balance
	toBefore := to.Balance

	// 更新余额
	from.Balance = from.Balance.Sub(amount)
	to.Balance = to.Balance.Add(amount)

	if err = tx.Model(&User{}).Where("id = ?", from.ID).Update("balance", from.Balance).Error; err != nil {
		return err
	}
	if err = tx.Model(&User{}).Where("id = ?", to.ID).Update("balance", to.Balance).Error; err != nil {
		return err
	}

	refId := uuid.New().String()

	outLog := WalletLog{
		UserID:    from.ID,
		BizKey:    bizKey,
		RefID:     refId,
		Direction: -1,
		Amount:    amount,
		Before:    fromBefore,
		After:     from.Balance,
		Reason:    ReasonTransfer,
		Meta:      datatypes.JSONMap{"role": "out", "from": fromUID, "to": toUID},
	}
	inLog := WalletLog{
		UserID:    to.ID,
		BizKey:    bizKey,
		RefID:     refId,
		Direction: 1,
		Amount:    amount,
		Before:    toBefore,
		After:     to.Balance,
		Reason:    ReasonTransfer,
		Meta:      datatypes.JSONMap{"role": "in", "from": fromUID, "to": toUID},
	}

	// —— 幂等策略（任选其一） ——
	// A) 直接 Create（若重复会报 1062 Duplicate，自行判断返回“已处理”）
	if err := tx.Create(&outLog).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return nil
		}
		return err
	}
	if err := tx.Create(&inLog).Error; err != nil {
		return err
	}

	// B) 或者：用 OnConflict.DoNothing（更干净）
	/*
	   if err := tx.Clauses(clause.OnConflict{
	       Columns:   []clause.Column{{Name: "user_id"}, {Name: "biz_key"}},
	       DoNothing: true,
	   }).Create(&outLog).Error; err != nil {
	       return err
	   }
	   if err := tx.Clauses(clause.OnConflict{
	       Columns:   []clause.Column{{Name: "user_id"}, {Name: "biz_key"}},
	       DoNothing: true,
	   }).Create(&inLog).Error; err != nil {
	       return err
	   }
	*/

	// 提交
	if err = tx.Commit().Error; err != nil {
		return err
	}
	// 事务成功
	return nil
}

func TestTransactionsLock() {
	db := New()
	var dbName string
	db.Raw("SELECT DATABASE()").Scan(&dbName)
	fmt.Println("当前 DATABASE() =", dbName)

	// err := db.AutoMigrate(&WalletLog{}) // 映射成数据库表结构
	// if err != nil {
	//	 fmt.Println("AutoMigrate err:", err)
	// }

	// 方式 1：函数式事务（推荐）GORM 在内部调用 db.Begin()，defer 检查 error 自动调用 Rollback() 或 Commit()
	// 在 panic 时也会回滚；是最安全、最干净的写法
	// fromUID, toUID, bizKey string, amount decimal.Decimal

	fromUid := "U1002"
	toUid := "U1001"
	amount, _ := decimal.NewFromString("160.08")
	// err := TransferTx(db, fromUid, toUid, "7191301982500556800", amount)
	// if err != nil {
	//	 fmt.Println("转账失败:", err)
	// }

	// 方式 2：显式事务（Begin / Commit / Rollback）
	err := TransferManual(db, fromUid, toUid, "7269222124152295424", amount)
	if err != nil {
		fmt.Println("转账失败:", err)
	}
}
