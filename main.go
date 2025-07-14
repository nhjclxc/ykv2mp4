package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
)

// 使用golang和ffmpeg实现优酷ykv视频解码为mp4
func main() {

	// 代码实现步骤：
	// 1、加载 `.ykv` 文件的全部二进制数据
	// 2、解码原ykv文件获取所有视频分片二级制数据。【查找所有 `"ftyp"` 的起始偏移，ftyp代表每个 MP4 分片开头，每一个分片就是一个mp4视频数据】
	// 3、遍历 offsets 切片，提取出每段数据并保存为 `part*.mp4`
	// 4、把每段的文件名写入 `filelist.txt` 中，用于记录待合并的分片路径
	// 5、执行FFmpeg命令合并分片mp4数据

	// 定义命令行参数
	inputFilePath := flag.String("inputFilePath", "video.ykv", "ykv文件地址（相对地址/绝对地址都可以）")
	filelistPath := flag.String("filelistPath", "filelist.txt", "切片文件地址输出路径")
	mergedOutputPath := flag.String("mergedOutputPath", "merged_output.mp4", "合并后的文件输出路径")
	ffmpegBinPath := flag.String("ffmpegBinPath", "", "ffmpeg的bin目录地址")

	// 自定义帮助信息
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "参数说明：")
		fmt.Fprintln(os.Stderr, "  -inputFilePath	string    ykv文件地址，默认读取：./video.ykv")
		fmt.Fprintln(os.Stderr, "  -filelistPath		string    切片文件地址输出路径，默认：./filelist.txt")
		fmt.Fprintln(os.Stderr, "  -mergedOutputPath	string    合并后的文件输出路径，默认：./merged_output.mp4")
		fmt.Fprintln(os.Stderr, "  -ffmpegBinPath	string    ffmpeg的bin目录地址，不传该参数则不会执行文件合并，注意'/'必须使用转移'//'")
	}

	// 解析命令行参数
	flag.Parse()

	// 打印读取结果
	fmt.Println("运行参数如下:")
	fmt.Println("ㅤㅤㅤykv文件地址:", *inputFilePath)
	fmt.Println("ㅤㅤㅤ切片文件地址输出路径:", *filelistPath)
	fmt.Println("ㅤㅤㅤ合并后的文件输出路径:", *mergedOutputPath)
	fmt.Println("ㅤㅤㅤffmpeg的bin目录地址:", *ffmpegBinPath)

	// 1、加载 `.ykv` 文件的全部二进制数据
	//inputFilePath := "video.ykv"
	//data, err := os.ReadFile(inputFilePath)
	data, err := os.ReadFile(*inputFilePath)
	if err != nil {
		fmt.Println("'video.ykv'文件不存在，请检查当前目录下是否有'video.ykv'文件！！！", err.Error())
		PrintLicenseNotice()
		return
	}

	// 2、解码原ykv文件获取所有视频分片二级制数据
	offsets := findFtypOffsets(data)
	if len(offsets) == 0 {
		fmt.Println("未找到任何 ftyp 片段")
		PrintLicenseNotice()
		return
	}
	fmt.Printf("共发现 %d 个 MP4 分片\n", len(offsets))

	// 补充最后一段终点为文件尾
	offsets = append(offsets, len(data))
	sort.Ints(offsets)

	// 创建 filelist.txt
	//filelistPath := "filelist.txt"
	//listFile, err := os.Create(filelistPath)
	listFile, err := os.Create(*filelistPath)
	if err != nil {
		fmt.Println("视频分片记录文件创建失败，请检测本工具是否对当前文件夹有写权限！！！", err.Error())
		PrintLicenseNotice()
		return
	}
	defer listFile.Close()

	// 3、遍历 offsets 切片，提取出每段数据并保存为 `part*.mp4`
	var mp4Files [][]byte = make([][]byte, len(offsets)-1, len(offsets)-1)
	for i := 0; i < len(offsets)-1; i++ {
		start := offsets[i]
		end := offsets[i+1]
		filename := fmt.Sprintf("part%d.mp4", i+1)

		mp4Files = append(mp4Files, data[start:end])
		err := os.WriteFile(filename, data[start:end], 0644)
		if err != nil {
			fmt.Println("视频分片失败，请检测本工具是否对当前文件夹有写权限！！！", err.Error())
			PrintLicenseNotice()
			return
		}

		fmt.Printf("✅ 提取 %s 成功，大小：%d 字节\n", filename, end-start)

		// 4、把每段的文件名写入 `filelist.txt` 中，用于记录待合并的分片路径
		// 写入到 filelist.txt
		_, err = listFile.WriteString(fmt.Sprintf("file '%s'\n", filename))
		if err != nil {
			fmt.Println("视频分片文件记录失败，请检测本工具是否对当前文件夹有写权限！！！", err.Error())
			PrintLicenseNotice()
			return
		}
	}

	fmt.Printf("生成 %s 完成，包含 %d 个文件\n", listFile.Name(), len(mp4Files))
	fmt.Println("\n🚀 所有分片已保存并生成 filelist.txt，即将运行ffmpeg进行文件合并。")
	//fmt.Println("ffmpeg -f concat -safe 0 -i filelist.txt -c copy full_output.mp4")

	// 5、执行FFmpeg命令合并分片mp4数据
	mergeMultMp4(mp4Files, listFile.Name(), *mergedOutputPath, *ffmpegBinPath)

	PrintLicenseNotice()

}

// PrintLicenseNotice 打印源码版权声明到控制台
func PrintLicenseNotice() {
	const (
		Bold    = "\033[1m"
		Red     = "\033[31m"
		BoldRed = "\033[1;31m"
		Reset   = "\033[0m"
	)
	fmt.Println("程序执行完毕，按 Enter 键退出...")
	fmt.Println("\n\n\n")

	fmt.Println(BoldRed + "📜 源码版权声明\n" + Reset)

	fmt.Println(BoldRed + "1. 本项目部分源代码由 ChatGPT 协助生成，作者对其进行了修改与整理。" + Reset)

	fmt.Println(BoldRed + "2. 据作者所知，截止当前（2025 年 7 月 13 日 12:30），GitHub 上尚无公开的关于 YKV 转码为 MP4 的完整 Golang 语言的开源实现。因此，欢迎学习与参考，但请注明项目来源 [github.com/nhjclxc/ykv2mp4](https://github.com/nhjclxc/ykv2mp4)，尊重原创。" + Reset)

	fmt.Println(BoldRed + "3. 本项目的源代码以 源代码开放（source-available） 形式发布，并遵循 Apache License 2.0 的大部分条款，但附加以下限制条款，对使用方式作出如下限制：" + Reset)
	fmt.Println(BoldRed + "   - 禁止将本项目全部或部分用于任何形式的直接或间接获利行为，包括但不限于：收费软件、订阅服务、SaaS 平台、广告变现、嵌入商业产品等。" + Reset)
	fmt.Println(BoldRed + "   - 如需用于商业用途或获取收益的场景，须事先取得作者书面授权。" + Reset)

	fmt.Println(BoldRed + "4. 本项目允许用于学习、研究、教学或个人非商业用途，前提是保留原始作者署名与此声明。\n" + Reset)

	fmt.Println(BoldRed + "📄 许可协议\n" + Reset)

	fmt.Println(BoldRed + "本项目以源代码开放（source-available）形式发布，基于 Apache License 2.0，并附加“禁止商用获利”限制条款。" + Reset)
	fmt.Println(BoldRed + "具体内容详见 LICENSE 文件。" + Reset)

	fmt.Println("\n\n\n")

	// 强制阻塞读取控制台输入
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}

// 读取每一个视频分片
// params：data原始的ykv文件数据
// return：返回每一个分别的起始位置
func findFtypOffsets(data []byte) []int {
	var offsets []int
	search := []byte("ftyp")
	i := 0
	for {
		index := bytes.Index(data[i:], search)
		if index == -1 {
			break
		}
		offset := i + index - 4 // 回溯4字节包含box length
		if offset >= 0 {
			offsets = append(offsets, offset)
		}
		i = i + index + 4
	}
	return offsets
}

// 合并多个MP4文件为一个
// params mp4Files：mp4二级制切片数据
// params listFileName：切片文件名
// params outputFile：合并后的文件输出地址
func mergeMultMp4(mp4Files [][]byte, listFileName string, outputFile string, ffmpegBinPath string) {

	// 检测是否有mp4文件
	if len(mp4Files) == 0 {
		log.Printf("当前目录未找到任何 *.mp4 数据")
		PrintLicenseNotice()
		return
	}
	if ffmpegBinPath == "" {
		log.Printf("未读取到ffmpeg的bin目录地址，ffmpegBinPath = %s，无法执行文件合并操作！！！\n", ffmpegBinPath)
		PrintLicenseNotice()
		return
	}

	// 调用 FFmpeg 执行合并
	//cmd := exec.Command("D:\\develop\\ffmpeg-2025-07-10-git-82aeee3c19-essentials_build\\ffmpeg-2025-07-10-git-82aeee3c19-essentials_build\\bin\\ffmpeg", "-f", "concat", "-safe", "0", "-i", listFileName, "-c", "copy", outputFile)
	cmd := exec.Command(ffmpegBinPath+"\\ffmpeg", "-f", "concat", "-safe", "0", "-i", listFileName, "-c", "copy", outputFile)

	// 输出 ffmpeg 运行日志到控制台
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("开始执行 ffmpeg 合并视频，请稍候...")
	err := cmd.Run()
	if err != nil {
		log.Printf("ffmpeg 执行失败: %v", err)
		PrintLicenseNotice()
	}

	fmt.Printf("合并完成，输出文件：%s\n", outputFile)
}

/*
运行参数如下:
ㅤㅤㅤykv文件地址: video.ykv
ㅤㅤㅤ切片文件地址输出路径: filelist.txt
ㅤㅤㅤ合并后的文件输出路径: merged_output.mp4
ㅤㅤㅤffmpeg的bin目录地址: D:\\develop\\ffmpeg-2025-07-10-git-82aeee3c19-essentials_build\\ffmpeg-2025-07-10-git-82aeee3c19-essentials_build\\bin
共发现 7 个 MP4 分片
✅ 提取 part1.mp4 成功，大小：51125482 字节
✅ 提取 part2.mp4 成功，大小：50638181 字节
✅ 提取 part3.mp4 成功，大小：54370259 字节
✅ 提取 part4.mp4 成功，大小：52553230 字节
✅ 提取 part5.mp4 成功，大小：55635110 字节
✅ 提取 part6.mp4 成功，大小：48883289 字节
✅ 提取 part7.mp4 成功，大小：43059133 字节
生成 filelist.txt 完成，包含 14 个文件

🚀 所有分片已保存并生成 filelist.txt，即将运行ffmpeg进行文件合并。

开始执行 ffmpeg 合并视频，请稍候...
ffmpeg version 2025-07-10-git-82aeee3c19-essentials_build-www.gyan.dev Copyright (c) 2000-2025 the FFmpeg developers
  built with gcc 15.1.0 (Rev6, Built by MSYS2 project)
  configuration: --enable-gpl --enable-version3 --enable-static --disable-w32threads --disable-autodetect --enable-fontconfig --enable-iconv --en
able-gnutls --enable-libxml2 --enable-gmp --enable-bzlib --enable-lzma --enable-zlib --enable-libsrt --enable-libssh --enable-libzmq --enable-avi
synth --enable-sdl2 --enable-libwebp --enable-libx264 --enable-libx265 --enable-libxvid --enable-libaom --enable-libopenjpeg --enable-libvpx --en
able-mediafoundation --enable-libass --enable-libfreetype --enable-libfribidi --enable-libharfbuzz --enable-libvidstab --enable-libvmaf --enable-
libzimg --enable-amf --enable-cuda-llvm --enable-cuvid --enable-dxva2 --enable-d3d11va --enable-d3d12va --enable-ffnvcodec --enable-libvpl --enab
le-nvdec --enable-nvenc --enable-vaapi --enable-openal --enable-libgme --enable-libopenmpt --enable-libopencore-amrwb --enable-libmp3lame --enabl
e-libtheora --enable-libvo-amrwbenc --enable-libgsm --enable-libopencore-amrnb --enable-libopus --enable-libspeex --enable-libvorbis --enable-librubberband
  libavutil      60.  4.101 / 60.  4.101
  libavcodec     62.  6.100 / 62.  6.100
  libavformat    62.  1.102 / 62.  1.102
  libavdevice    62.  0.100 / 62.  0.100
  libavfilter    11.  1.100 / 11.  1.100
  libswscale      9.  0.100 /  9.  0.100
  libswresample   6.  0.100 /  6.  0.100
[mov,mp4,m4a,3gp,3g2,mj2 @ 000001fbc52c5f80] Auto-inserting h264_mp4toannexb bitstream filter
Input #0, concat, from 'filelist.txt':
  Duration: N/A, start: 0.000000, bitrate: 2044 kb/s
  Stream #0:0(und): Video: h264 (High) (avc1 / 0x31637661), yuv420p(tv, bt709, progressive), 1280x720 [SAR 1:1 DAR 16:9], 1916 kb/s, 25 fps, 25 tbr, 12800 tbn
    Metadata:
      handler_name    : VideoHandler
      vendor_id       : [0][0][0][0]
  Stream #0:1(und): Audio: aac (LC) (mp4a / 0x6134706D), 44100 Hz, stereo, fltp, 128 kb/s
    Metadata:
      handler_name    : SoundHandler
      vendor_id       : [0][0][0][0]
Stream mapping:
  Stream #0:0 -> #0:0 (copy)
  Stream #0:1 -> #0:1 (copy)
Output #0, mp4, to 'merged_output.mp4':
  Metadata:
    encoder         : Lavf62.1.102
  Stream #0:0(und): Video: h264 (High) (avc1 / 0x31637661), yuv420p(tv, bt709, progressive), 1280x720 [SAR 1:1 DAR 16:9], q=2-31, 1916 kb/s, 25 fps, 25 tbr, 12800 tbn
    Metadata:
      handler_name    : VideoHandler
      vendor_id       : [0][0][0][0]
  Stream #0:1(und): Audio: aac (LC) (mp4a / 0x6134706D), 44100 Hz, stereo, fltp, 128 kb/s
    Metadata:
      handler_name    : SoundHandler
      vendor_id       : [0][0][0][0]
Press [q] to stop, [?] for help
[mov,mp4,m4a,3gp,3g2,mj2 @ 000001fbc52d0d00] Auto-inserting h264_mp4toannexb bitstream filter
    Last message repeated 3 times
[mov,mp4,m4a,3gp,3g2,mj2 @ 000001fbc52d0d00] Auto-inserting h264_mp4toannexb bitstream filtere+03x elapsed=0:00:00.51
    Last message repeated 1 times
[out#0/mp4 @ 000001fbc52d8b00] video:325559KiB audio:21271KiB subtitle:0KiB other streams:0KiB global headers:0KiB muxing overhead: 0.293970%
frame=34033 fps=0.0 q=-1.0 Lsize=  347849KiB time=00:22:41.28 bitrate=2093.3kbits/s speed=1.56e+03x elapsed=0:00:00.87
合并完成，输出文件：merged_output.mp4


*/
