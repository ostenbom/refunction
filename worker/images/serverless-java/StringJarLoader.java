import java.util.Base64;
import java.util.zip.ZipInputStream;
import java.util.zip.ZipEntry;
import java.io.InputStream;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;

public class StringJarLoader extends ClassLoader {
    private String jarString;
    public StringJarLoader(String jarString) {
        super();
        this.jarString = jarString;
    }

    @Override
    public Class<?> findClass(String name) {
        byte[] jarBytes = Base64.getDecoder().decode(this.jarString);
        InputStream byteStream = new ByteArrayInputStream(jarBytes);
        ZipInputStream zipStream = new ZipInputStream(byteStream);
        ZipEntry entry;
        try {
        while ((entry = zipStream.getNextEntry()) != null) {
            System.out.println(entry.getName());
            if (entry.getName().equals(name + ".class")) {
                int size;
                byte[] buffer = new byte[2048];
                ByteArrayOutputStream bos = new ByteArrayOutputStream(buffer.length);
                while ((size = zipStream.read(buffer, 0, buffer.length)) != -1) {
                    bos.write(buffer, 0, size);
                }

                byte[] bt = bos.toByteArray();
                return defineClass(name, bt, 0, bt.length);
            }
        }
        } catch(Exception e) {
            System.out.println(e);
            return null;
        }

        return null;
    }

}
