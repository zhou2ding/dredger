CREATE TABLE `soil_regions` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `x_min` DOUBLE NOT NULL,
  `x_max` DOUBLE NOT NULL,
  `y_min` DOUBLE NOT NULL,
  `y_max` DOUBLE NOT NULL,
  `z_min` DOUBLE NOT NULL,
  `z_max` DOUBLE NOT NULL,
  `soil_type` VARCHAR(255) NOT NULL COMMENT '土质',
  INDEX `idx_spatial_query` (`x_min`, `x_max`, `y_min`, `y_max`) -- 关键：创建复合索引以加速查询
) COMMENT='土质区域划分表';